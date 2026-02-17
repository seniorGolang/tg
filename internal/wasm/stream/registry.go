// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

type StreamState struct {
	Reader io.Reader
	Writer io.Writer

	ReadBufferPtr  uint32
	WriteBufferPtr uint32
	BufferSize     uint32

	ReadBufferDataSize  uint32
	WriteBufferDataSize uint32

	Closed bool
	Mu     sync.RWMutex

	ReadBuffer  []byte
	WriteBuffer []byte

	readerDone sync.WaitGroup
}

type Registry struct {
	streams map[uint32]*StreamState
	nextID  uint32
	mu      sync.RWMutex
}

type RingBufferReader interface {
	ReadFromRingBuffer(ctx context.Context, h memory.Host, bufferPtr uint32, dataSize uint32, data []byte) (read int, err error)
}

type RingBufferWriter interface {
	WriteToRingBuffer(ctx context.Context, h memory.Host, bufferPtr uint32, dataSize uint32, data []byte) (written int, err error)
}

func NewStreamRegistry() (registry *Registry) {
	return &Registry{
		streams: make(map[uint32]*StreamState),
		nextID:  1,
	}
}

func (r *Registry) NewStream(ctx context.Context, h memory.Host, reader io.Reader, writer io.Writer, bufferSize uint32) (streamID uint32, err error) {

	if bufferSize == 0 {
		bufferSize = DefaultRingBufferSize
	}

	var readBufferPtr uint32
	if readBufferPtr, err = CreateRingBuffer(ctx, h, bufferSize); err != nil {
		return 0, fmt.Errorf("failed to create read ring buffer: %w", err)
	}

	var writeBufferPtr uint32
	if writeBufferPtr, err = CreateRingBuffer(ctx, h, bufferSize); err != nil {
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
		ReadBufferDataSize:  bufferSize,
		WriteBufferDataSize: bufferSize,
		Closed:              false,
		ReadBuffer:          make([]byte, 4096),
		WriteBuffer:         make([]byte, 4096),
	}

	r.streams[streamID] = streamState

	return
}

func (r *Registry) GetStreamState(streamID uint32) (state *StreamState, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	s := r.streams[streamID]
	return s, s != nil
}

func (r *Registry) CloseStream(ctx context.Context, h memory.Host, streamID uint32) {

	r.mu.Lock()
	stream := r.streams[streamID]
	if stream == nil {
		r.mu.Unlock()
		return
	}

	delete(r.streams, streamID)
	r.mu.Unlock()

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

	if wasClosed {
		return
	}

	if readBufferPtr != 0 {
		_ = SetClosed(ctx, h, readBufferPtr)
	}
	if writeBufferPtr != 0 {
		_ = SetClosed(ctx, h, writeBufferPtr)
	}

	if readBufferPtr != 0 {
		DestroyRingBuffer(ctx, h, readBufferPtr)
	}
	if writeBufferPtr != 0 {
		DestroyRingBuffer(ctx, h, writeBufferPtr)
	}
}

func (r *Registry) GetBufferPtr(streamID uint32) (bufferPtr uint32, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[streamID]
	if stream == nil {
		return
	}

	return stream.ReadBufferPtr, true
}

func (r *Registry) GetReadBufferPtr(streamID uint32) (bufferPtr uint32, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[streamID]
	if stream == nil {
		return
	}

	return stream.ReadBufferPtr, true
}

func (r *Registry) GetWriteBufferPtr(streamID uint32) (bufferPtr uint32, ok bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[streamID]
	if stream == nil {
		return
	}

	return stream.WriteBufferPtr, true
}

func (r *Registry) WaitReaderDone(streamID uint32) {

	r.mu.RLock()
	stream := r.streams[streamID]
	r.mu.RUnlock()

	if stream != nil {
		stream.readerDone.Wait()
	}
}

type registry interface {
	NewStream(ctx context.Context, h memory.Host, reader io.Reader, writer io.Writer, bufferSize uint32) (streamID uint32, err error)
	GetStreamState(streamID uint32) (state *StreamState, ok bool)
	CloseStream(ctx context.Context, h memory.Host, streamID uint32)
	GetBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	GetReadBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	GetWriteBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	WaitReaderDone(streamID uint32)
}
