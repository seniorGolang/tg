// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/stream"
)

// connState хранит состояние соединения.
type connState struct {
	conn       net.Conn
	reader     *bufio.Reader // Для TLS соединений используем bufio.Reader для поддержки Peek
	isTLS      bool
	isClosed   bool
	streamID   uint32 // ID потока для стриминга через кольцевой буфер
	streamCtx  context.Context
	streamHost *host.Host
}

type netManager struct {
	connLock     sync.RWMutex
	connID       uint64
	connMap      map[uint64]net.Conn
	connStateMap map[uint64]*connState
	listenerLock sync.RWMutex
	listenerID   uint64
	listenerMap  map[uint64]net.Listener

	// onNewConnectionCallback - функция для вызова callback on_new_connection в WASM
	// Вызывается автоматически при принятии нового соединения через ListenerAccept
	// Если nil, callback не вызывается
	onNewConnectionCallback func(connID uint64)
}

func NewNetManager() (nm *netManager) {
	return &netManager{
		connMap:                 make(map[uint64]net.Conn),
		connStateMap:            make(map[uint64]*connState),
		listenerMap:             make(map[uint64]net.Listener),
		onNewConnectionCallback: nil,
	}
}

// SetOnNewConnectionCallback: callback вызывается при принятии соединения через ListenerAccept.
func (nm *netManager) SetOnNewConnectionCallback(callback func(connID uint64)) {

	nm.listenerLock.Lock()
	defer nm.listenerLock.Unlock()

	nm.onNewConnectionCallback = callback
}

func (nm *netManager) GetConn(connID uint64) (conn net.Conn, err error) {

	nm.connLock.RLock()
	defer nm.connLock.RUnlock()

	var ok bool
	if conn, ok = nm.connMap[connID]; !ok {
		return nil, fmt.Errorf(i18n.Msg("connection id %d does not exist"), connID)
	}

	return
}

func (nm *netManager) StoreConnWithStream(ctx context.Context, h any, conn net.Conn) (connID uint64) {

	nm.connLock.Lock()
	defer nm.connLock.Unlock()

	nm.connID++
	connID = nm.connID
	nm.connMap[connID] = conn

	var reader *bufio.Reader
	var tlsConn *tls.Conn
	var ok bool
	isTLS := false
	if tlsConn, ok = conn.(*tls.Conn); ok {
		isTLS = true
		// Для TLS соединений создаем bufio.Reader для поддержки Peek (как в net/http)
		reader = bufio.NewReader(tlsConn)
	}

	var streamID uint32
	var streamHost *host.Host
	if ctx != nil && h != nil {
		var hostOK bool
		var hst *host.Host
		if hst, hostOK = h.(*host.Host); hostOK && hst != nil && hst.StreamRegistry != nil {
			if sid, err := hst.StreamRegistry.NewStream(ctx, hst, conn, conn, 0); err == nil {
				streamID = sid
				streamHost = hst

				var streamOK bool
				var streamReg *stream.Registry
				if streamReg, streamOK = hst.StreamRegistry.(*stream.Registry); streamOK && streamReg != nil {
					streamReader := stream.NewStreamReader(ctx, sid, streamReg, hst)
					streamWriter := stream.NewStreamWriter(ctx, sid, streamReg, hst)

					streamReader.StartReader()
					streamWriter.StartWriter()
				}
			}
		}
	}

	nm.connStateMap[connID] = &connState{
		conn:       conn,
		reader:     reader,
		isTLS:      isTLS,
		isClosed:   false,
		streamID:   streamID,
		streamCtx:  ctx,
		streamHost: streamHost,
	}

	return
}

func (nm *netManager) DelConn(connID uint64) {

	nm.connLock.Lock()
	conn, ok := nm.connMap[connID]
	state := nm.connStateMap[connID]
	if ok {
		delete(nm.connMap, connID)
		delete(nm.connStateMap, connID)
	}
	nm.connLock.Unlock()

	if !ok {
		return
	}

	// Риск: часть error-cleanup путей вызывает DelConn напрямую, минуя connClose.
	// Поэтому DelConn сам освобождает связанный stream, чтобы не держать ring buffers
	// в WASM памяти. CloseStream идемпотентен для отсутствующего stream, поэтому
	// повторный cleanup после обычного connClose не должен ломать caller.
	if state != nil && state.streamID != 0 && state.streamHost != nil && state.streamHost.StreamRegistry != nil {
		streamCtx := state.streamCtx
		if streamCtx == nil {
			streamCtx = context.Background()
		}
		state.streamHost.StreamRegistry.CloseStream(streamCtx, state.streamHost, state.streamID)
	}

	go func() {
		_ = conn.Close()
	}()
}

func (nm *netManager) GetConnState(connID uint64) (state *connState, err error) {

	nm.connLock.RLock()
	defer nm.connLock.RUnlock()

	var ok bool
	if state, ok = nm.connStateMap[connID]; !ok {
		return nil, fmt.Errorf(i18n.Msg("connection state for id %d does not exist"), connID)
	}

	return
}

func (nm *netManager) GetReader(connID uint64) (reader *bufio.Reader, err error) {

	var state *connState
	if state, err = nm.GetConnState(connID); err != nil {
		return
	}

	return state.reader, nil
}

// MarkConnClosed помечает соединение как закрытое.
func (nm *netManager) MarkConnClosed(connID uint64) {

	nm.connLock.Lock()
	defer nm.connLock.Unlock()

	if state, ok := nm.connStateMap[connID]; ok {
		state.isClosed = true
	}
}

func (nm *netManager) GetListener(listenerID uint64) (listener net.Listener, err error) {

	nm.listenerLock.RLock()
	defer nm.listenerLock.RUnlock()

	var ok bool
	if listener, ok = nm.listenerMap[listenerID]; !ok {
		return nil, fmt.Errorf(i18n.Msg("listener id %d does not exist"), listenerID)
	}

	return
}

// StoreListener сохраняет слушатель и возвращает его ID.
func (nm *netManager) StoreListener(listener net.Listener) (listenerID uint64) {

	nm.listenerLock.Lock()
	defer nm.listenerLock.Unlock()

	nm.listenerID++
	nm.listenerMap[nm.listenerID] = listener

	return nm.listenerID
}

// DelListener удаляет слушатель по ID и закрывает его.
func (nm *netManager) DelListener(listenerID uint64) {

	nm.listenerLock.Lock()
	defer nm.listenerLock.Unlock()

	listener, ok := nm.listenerMap[listenerID]
	if ok {
		delete(nm.listenerMap, listenerID)
		// Закрываем слушатель асинхронно, чтобы не блокировать мьютекс
		go func() {
			_ = listener.Close()
		}()
	}
}
