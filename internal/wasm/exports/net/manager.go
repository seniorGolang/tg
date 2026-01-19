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
	conn     net.Conn
	reader   *bufio.Reader // Для TLS соединений используем bufio.Reader для поддержки Peek
	isTLS    bool
	isClosed bool
	streamID uint32 // ID потока для стриминга через кольцевой буфер
}

// netManager управляет сетевыми соединениями и слушателями.
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

	conn, ok := nm.connMap[connID]
	if !ok {
		return nil, fmt.Errorf(i18n.Msg("connection id %d does not exist"), connID)
	}

	return
}

// StoreConn сохраняет соединение и возвращает его ID.
// Если h и streamRegistry не nil, создает кольцевой буфер для стриминга.
func (nm *netManager) StoreConn(conn net.Conn) (connID uint64) {
	return nm.StoreConnWithStream(context.Background(), nil, conn)
}

// StoreConnWithStream сохраняет соединение с поддержкой стриминга через кольцевой буфер.
func (nm *netManager) StoreConnWithStream(ctx context.Context, h any, conn net.Conn) (connID uint64) {

	nm.connLock.Lock()
	defer nm.connLock.Unlock()

	nm.connID++
	connID = nm.connID
	nm.connMap[connID] = conn

	// Определяем, является ли соединение TLS
	var reader *bufio.Reader
	isTLS := false
	if tlsConn, ok := conn.(*tls.Conn); ok {
		isTLS = true
		// Для TLS соединений создаем bufio.Reader для поддержки Peek (как в net/http)
		reader = bufio.NewReader(tlsConn)
	}

	var streamID uint32
	// Создаем кольцевой буфер для стриминга, если доступны Host и StreamRegistry
	if ctx != nil && h != nil {
		if host, ok := h.(*host.Host); ok && host != nil && host.StreamRegistry != nil {
			if sid, err := host.StreamRegistry.NewStream(ctx, host, conn, conn, 0); err == nil {
				streamID = sid

				// Запускаем горутины для синхронизации данных между net.Conn и кольцевым буфером
				// Используем StreamReader и StreamWriter для синхронизации
				if streamReg, ok := host.StreamRegistry.(*stream.StreamRegistry); ok && streamReg != nil {
					streamReader := stream.NewStreamReader(ctx, sid, streamReg, host)
					streamWriter := stream.NewStreamWriter(ctx, sid, streamReg, host)

					// Запускаем горутины для чтения/записи
					streamReader.StartReader()
					streamWriter.StartWriter()
				}
			}
		}
	}

	nm.connStateMap[connID] = &connState{
		conn:     conn,
		reader:   reader,
		isTLS:    isTLS,
		isClosed: false,
		streamID: streamID,
	}

	return
}

// DelConn удаляет соединение по ID и закрывает его.
func (nm *netManager) DelConn(connID uint64) {

	nm.connLock.Lock()
	defer nm.connLock.Unlock()

	conn, ok := nm.connMap[connID]
	if ok {
		delete(nm.connMap, connID)
		delete(nm.connStateMap, connID)
		// Закрываем соединение асинхронно, чтобы не блокировать мьютекс
		go func() {
			_ = conn.Close()
		}()
	}
}

func (nm *netManager) GetConnState(connID uint64) (state *connState, err error) {

	nm.connLock.RLock()
	defer nm.connLock.RUnlock()

	state, ok := nm.connStateMap[connID]
	if !ok {
		return nil, fmt.Errorf(i18n.Msg("connection state for id %d does not exist"), connID)
	}

	return
}

func (nm *netManager) GetReader(connID uint64) (reader *bufio.Reader, err error) {

	state, err := nm.GetConnState(connID)
	if err != nil {
		return nil, err
	}

	reader = state.reader
	return
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

	listener, ok := nm.listenerMap[listenerID]
	if !ok {
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

	listenerID = nm.listenerID
	return
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
