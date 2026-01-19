// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// StreamState хранит состояние потока данных.
type StreamState struct {
	// Источники данных на хосте
	Reader io.Reader
	Writer io.Writer

	// Кольцевые буферы в WASM памяти
	// ReadBufferPtr - буфер для чтения (хост → WASM): хост записывает, WASM читает
	// WriteBufferPtr - буфер для записи (WASM → хост): WASM записывает, хост читает
	ReadBufferPtr  uint32 // Указатель на буфер для чтения (хост → WASM)
	WriteBufferPtr uint32 // Указатель на буфер для записи (WASM → хост)
	BufferSize     uint32 // Размер буфера (включая заголовок)

	// Кэшированные размеры областей данных (константы, не изменяются после создания)
	ReadBufferDataSize  uint32 // Размер области данных ReadBuffer (BufferSize - HeaderSize)
	WriteBufferDataSize uint32 // Размер области данных WriteBuffer (BufferSize - HeaderSize)

	// Состояние
	Closed bool
	Mu     sync.RWMutex

	// Временные буферы хоста
	ReadBuffer  []byte // Буфер для чтения из Reader
	WriteBuffer []byte // Буфер для записи в Writer

	// Синхронизация завершения StartReader()
	// readerDone сигнализирует, что StartReader() завершился и записал все данные
	readerDone sync.WaitGroup
}

// StreamRegistry управляет потоками данных.
type StreamRegistry struct {
	streams map[uint32]*StreamState
	nextID  uint32
	mu      sync.RWMutex
}

// RingBufferReader интерфейс для чтения из кольцевого буфера.
type RingBufferReader interface {
	ReadFromRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32, dataSize uint32, data []byte) (read int, err error)
}

// RingBufferWriter интерфейс для записи в кольцевой буфер.
type RingBufferWriter interface {
	WriteToRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32, dataSize uint32, data []byte) (written int, err error)
}

func NewStreamRegistry() (registry *StreamRegistry) {

	return &StreamRegistry{
		streams: make(map[uint32]*StreamState),
		nextID:  1,
	}
}

// NewStream выделяет два кольцевых буфера в WASM: чтение (хост → WASM) и запись (WASM → хост).
func (r *StreamRegistry) NewStream(ctx context.Context, h *host.Host, reader io.Reader, writer io.Writer, bufferSize uint32) (streamID uint32, err error) {

	if bufferSize == 0 {
		bufferSize = DefaultRingBufferSize
	}

	// Выделяем два кольцевых буфера в WASM памяти
	// ReadBufferPtr - для чтения (хост → WASM): хост записывает, WASM читает
	// WriteBufferPtr - для записи (WASM → хост): WASM записывает, хост читает
	var readBufferPtr uint32
	if readBufferPtr, err = CreateRingBuffer(ctx, h, bufferSize); err != nil {
		return 0, fmt.Errorf("failed to create read ring buffer: %w", err)
	}

	var writeBufferPtr uint32
	if writeBufferPtr, err = CreateRingBuffer(ctx, h, bufferSize); err != nil {
		// Освобождаем readBufferPtr при ошибке
		DestroyRingBuffer(ctx, h, readBufferPtr)
		return 0, fmt.Errorf("failed to create write ring buffer: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	streamID = r.nextID
	r.nextID++

	streamState := &StreamState{
		Reader:              reader,
		Writer:              writer,
		ReadBufferPtr:       readBufferPtr,
		WriteBufferPtr:      writeBufferPtr,
		BufferSize:          bufferSize + RingBufferHeaderSize,
		ReadBufferDataSize:  bufferSize, // Кэшируем размер области данных (константа)
		WriteBufferDataSize: bufferSize, // Кэшируем размер области данных (константа)
		Closed:              false,
		ReadBuffer:          make([]byte, 4096), // 4KB буфер для чтения
		WriteBuffer:         make([]byte, 4096), // 4KB буфер для записи
	}

	r.streams[streamID] = streamState

	return
}

func (r *StreamRegistry) GetStream(streamID uint32) (state any) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	state = r.streams[streamID]
	return
}

func (r *StreamRegistry) GetStreamState(streamID uint32) (state any) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	state = r.streams[streamID]
	return
}

// GetStreamStateDirect эффективнее GetStreamState (без type assertion).
func (r *StreamRegistry) GetStreamStateDirect(streamID uint32) (state *StreamState, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	state = r.streams[streamID]
	ok = state != nil
	return
}

func (r *StreamRegistry) CloseStream(ctx context.Context, h *host.Host, streamID uint32) {

	r.mu.Lock()
	stream := r.streams[streamID]
	if stream == nil {
		r.mu.Unlock()
		return
	}

	// Удаляем из реестра сразу, чтобы избежать повторного закрытия
	delete(r.streams, streamID)
	r.mu.Unlock()

	// Закрываем поток вне блокировки реестра
	stream.Mu.Lock()
	wasClosed := stream.Closed
	stream.Closed = true
	readBufferPtr := stream.ReadBufferPtr
	writeBufferPtr := stream.WriteBufferPtr
	stream.Reader = nil
	stream.Writer = nil
	stream.ReadBuffer = nil
	stream.WriteBuffer = nil
	stream.Mu.Unlock()

	// Если уже был закрыт, не делаем повторных операций
	if wasClosed {
		return
	}

	if readBufferPtr != 0 {
		_ = SetClosed(ctx, h, readBufferPtr)
	}
	if writeBufferPtr != 0 {
		_ = SetClosed(ctx, h, writeBufferPtr)
	}

	// Освобождаем кольцевые буферы
	if readBufferPtr != 0 {
		DestroyRingBuffer(ctx, h, readBufferPtr)
	}
	if writeBufferPtr != 0 {
		DestroyRingBuffer(ctx, h, writeBufferPtr)
	}
}

// GetBufferPtr для обратной совместимости возвращает ReadBufferPtr (хост → WASM).
func (r *StreamRegistry) GetBufferPtr(streamID uint32) (bufferPtr uint32, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[streamID]
	if stream == nil {
		return
	}

	bufferPtr = stream.ReadBufferPtr
	ok = true
	return
}

func (r *StreamRegistry) GetReadBufferPtr(streamID uint32) (bufferPtr uint32, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[streamID]
	if stream == nil {
		return
	}

	bufferPtr = stream.ReadBufferPtr
	ok = true
	return
}

func (r *StreamRegistry) GetWriteBufferPtr(streamID uint32) (bufferPtr uint32, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[streamID]
	if stream == nil {
		return
	}

	bufferPtr = stream.WriteBufferPtr
	ok = true
	return
}

// WaitReaderDone синхронизирует завершение StartReader() перед установкой Closed.
func (r *StreamRegistry) WaitReaderDone(streamID uint32) {

	r.mu.RLock()
	stream := r.streams[streamID]
	r.mu.RUnlock()

	if stream != nil {
		stream.readerDone.Wait()
	}
}
