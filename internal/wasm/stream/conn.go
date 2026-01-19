// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// StreamConn оборачивает net.Conn через стриминг с кольцевым буфером.
// StreamConn: WASM — кольцевой буфер в памяти; хост — net.Conn, синхронизация с буфером.
type StreamConn struct {
	conn     net.Conn
	streamID uint32
	registry *StreamRegistry
	host     *host.Host
	ctx      context.Context

	// Горутины для чтения/записи
	readDone  chan error
	writeDone chan error
	once      sync.Once
	closed    bool
	mu        sync.RWMutex

	// Deadlines
	readDeadline  time.Time
	writeDeadline time.Time
	readMu        sync.RWMutex
	writeMu       sync.RWMutex

	// Каналы для уведомления о готовности буфера (буферизованные, размер 1)
	readReady  chan struct{}
	writeReady chan struct{}
}

// notifyReadReady безопасно отправляет уведомление о готовности к чтению (non-blocking).
func (sc *StreamConn) notifyReadReady() {

	select {
	case sc.readReady <- struct{}{}:
		// Уведомление отправлено
	default:
		// Канал уже имеет уведомление, не блокируем
	}
}

// notifyWriteReady безопасно отправляет уведомление о готовности к записи (non-blocking).
func (sc *StreamConn) notifyWriteReady() {

	select {
	case sc.writeReady <- struct{}{}:
		// Уведомление отправлено
	default:
		// Канал уже имеет уведомление, не блокируем
	}
}

func NewStreamConn(ctx context.Context, conn net.Conn, streamID uint32, registry *StreamRegistry, h *host.Host) (streamConn *StreamConn) {

	sc := &StreamConn{
		conn:       conn,
		streamID:   streamID,
		registry:   registry,
		host:       h,
		ctx:        ctx,
		readDone:   make(chan error, 1),
		writeDone:  make(chan error, 1),
		readReady:  make(chan struct{}, 1),
		writeReady: make(chan struct{}, 1),
	}

	// Запускаем горутины для чтения/записи
	sc.start()

	streamConn = sc
	return
}

// start запускает горутины для чтения и записи.
func (sc *StreamConn) start() {

	state, ok := sc.registry.GetStreamStateDirect(sc.streamID)
	if !ok || state == nil {
		return
	}

	// Горутина для чтения из conn и записи в кольцевой буфер
	go sc.readLoop(state)

	// Горутина для чтения из кольцевого буфера и записи в conn
	go sc.writeLoop(state)
}

func (sc *StreamConn) readLoop(state *StreamState) {

	buffer := make([]byte, 4096)
	for {
		// Проверяем контекст
		select {
		case <-sc.ctx.Done():
			sc.readDone <- sc.ctx.Err()
			return
		default:
		}

		// Проверяем deadline для чтения
		sc.readMu.RLock()
		deadline := sc.readDeadline
		sc.readMu.RUnlock()

		if !deadline.IsZero() && time.Now().After(deadline) {
			sc.readDone <- io.EOF
			return
		}

		// Устанавливаем deadline на conn
		if !deadline.IsZero() {
			_ = sc.conn.SetReadDeadline(deadline)
		}

		// Читаем данные из conn
		n, err := sc.conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// Закрываем поток при EOF
				sc.registry.CloseStream(sc.ctx, sc.host, sc.streamID)
			}
			sc.readDone <- err
			return
		}

		if n == 0 {
			continue
		}

		// Записываем данные в кольцевой буфер для чтения (хост → WASM)
		// Блокируем до полной записи всех данных, если буфер полон
		_, writeErr := writeToRingBufferWithRetry(sc.ctx, sc.host, state.ReadBufferPtr, state.ReadBufferDataSize, buffer[:n], sc.readReady)
		if writeErr != nil {
			if writeErr == io.EOF {
				sc.registry.CloseStream(sc.ctx, sc.host, sc.streamID)
			}
			sc.readDone <- writeErr
			return
		}
	}
}

func (sc *StreamConn) writeLoop(state *StreamState) {

	buffer := make([]byte, 4096)
	for {
		// Проверяем контекст
		select {
		case <-sc.ctx.Done():
			sc.writeDone <- sc.ctx.Err()
			return
		default:
		}

		// Проверяем deadline для записи
		sc.writeMu.RLock()
		deadline := sc.writeDeadline
		sc.writeMu.RUnlock()

		if !deadline.IsZero() && time.Now().After(deadline) {
			sc.writeDone <- io.EOF
			return
		}

		// Устанавливаем deadline на conn
		if !deadline.IsZero() {
			_ = sc.conn.SetWriteDeadline(deadline)
		}

		// Читаем данные из кольцевого буфера для записи (WASM → хост)
		n, err := readFromRingBufferWithRetry(sc.ctx, sc.host, state.WriteBufferPtr, state.WriteBufferDataSize, buffer, sc.writeReady)
		if err != nil {
			if err == io.EOF {
				sc.registry.CloseStream(sc.ctx, sc.host, sc.streamID)
			}
			sc.writeDone <- err
			return
		}

		// Записываем данные в conn
		// Блокируем до полной записи всех данных
		totalWritten := 0
		for totalWritten < n {
			var writtenBytes int
			var writeErr error
			if writtenBytes, writeErr = sc.conn.Write(buffer[totalWritten:n]); writeErr != nil {
				sc.writeDone <- writeErr
				return
			}

			if writtenBytes == 0 {
				// Conn не может записать данные, это нормально для некоторых типов соединений
				// Продолжаем попытку записи в следующей итерации
				// Если это критично, можно добавить канал уведомлений для conn
				continue
			}

			totalWritten += writtenBytes
		}
	}
}

// Read реализует net.Conn.Read.
// WASM должен читать данные из кольцевого буфера напрямую.
func (sc *StreamConn) Read(buffer []byte) (n int, err error) {

	sc.mu.RLock()
	closed := sc.closed
	sc.mu.RUnlock()

	if closed {
		return 0, io.ErrClosedPipe
	}

	// Для чтения WASM должен использовать кольцевой буфер напрямую
	// Эта функция может быть использована для проверки состояния
	state, ok := sc.registry.GetStreamStateDirect(sc.streamID)
	if !ok || state == nil {
		return 0, io.ErrClosedPipe
	}

	// Читаем данные из кольцевого буфера для чтения (хост → WASM)
	n, err = ReadFromRingBuffer(sc.ctx, sc.host, state.ReadBufferPtr, state.ReadBufferDataSize, buffer)

	// Уведомляем о том, что место в буфере освободилось
	if n > 0 {
		sc.notifyReadReady()
	}

	return
}

// Write реализует net.Conn.Write.
// WASM должен записывать данные в кольцевой буфер напрямую.
func (sc *StreamConn) Write(buffer []byte) (n int, err error) {

	sc.mu.RLock()
	closed := sc.closed
	sc.mu.RUnlock()

	if closed {
		return 0, io.ErrClosedPipe
	}

	// Для записи WASM должен использовать кольцевой буфер напрямую
	// Эта функция может быть использована для проверки состояния
	state, ok := sc.registry.GetStreamStateDirect(sc.streamID)
	if !ok || state == nil {
		return 0, io.ErrClosedPipe
	}

	// Записываем данные в кольцевой буфер для записи (WASM → хост)
	n, err = WriteToRingBuffer(sc.ctx, sc.host, state.WriteBufferPtr, state.WriteBufferDataSize, buffer)

	// Уведомляем о том, что данные появились в буфере
	if n > 0 {
		sc.notifyWriteReady()
	}

	return
}

// Close реализует net.Conn.Close.
func (sc *StreamConn) Close() (err error) {

	sc.once.Do(func() {
		sc.mu.Lock()
		sc.closed = true
		sc.mu.Unlock()

		// Закрываем поток
		sc.registry.CloseStream(sc.ctx, sc.host, sc.streamID)

		// Закрываем conn
		_ = sc.conn.Close()
	})

	return nil
}

// LocalAddr реализует net.Conn.LocalAddr.
func (sc *StreamConn) LocalAddr() (addr net.Addr) {

	return sc.conn.LocalAddr()
}

// RemoteAddr реализует net.Conn.RemoteAddr.
func (sc *StreamConn) RemoteAddr() (addr net.Addr) {

	return sc.conn.RemoteAddr()
}

// SetDeadline реализует net.Conn.SetDeadline.
func (sc *StreamConn) SetDeadline(deadline time.Time) (err error) {

	sc.readMu.Lock()
	sc.readDeadline = deadline
	sc.readMu.Unlock()

	sc.writeMu.Lock()
	sc.writeDeadline = deadline
	sc.writeMu.Unlock()

	return sc.conn.SetDeadline(deadline)
}

// SetReadDeadline реализует net.Conn.SetReadDeadline.
func (sc *StreamConn) SetReadDeadline(deadline time.Time) (err error) {

	sc.readMu.Lock()
	sc.readDeadline = deadline
	sc.readMu.Unlock()

	return sc.conn.SetReadDeadline(deadline)
}

// SetWriteDeadline реализует net.Conn.SetWriteDeadline.
func (sc *StreamConn) SetWriteDeadline(deadline time.Time) (err error) {

	sc.writeMu.Lock()
	sc.writeDeadline = deadline
	sc.writeMu.Unlock()

	return sc.conn.SetWriteDeadline(deadline)
}
