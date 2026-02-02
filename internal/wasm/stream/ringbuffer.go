// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

const (
	RingBufferHeaderSize  = 20
	DefaultRingBufferSize = 64 * 1024
	MinRingBufferSize     = 4 * 1024
	MaxRingBufferSize     = 1024 * 1024
)

// RingBufferHeader представляет заголовок кольцевого буфера в WASM памяти.
// Структура в памяти (little-endian):
// [0-3]   BufferSize uint32  // Общий размер буфера (включая заголовок)
// [4-7]   DataSize uint32    // Размер области данных (BufferSize - HeaderSize)
// [8-11]  ReadIndex uint32   // Индекс чтения (атомарный)
// [12-15] WriteIndex uint32  // Индекс записи (атомарный)
// [16-19] Closed uint32      // Флаг закрытия (0 = открыт, 1 = закрыт)
type RingBufferHeader struct {
	BufferSize uint32
	DataSize   uint32
	ReadIndex  uint32
	WriteIndex uint32
	Closed     uint32
}

func RingBufferOffset() uint32 {
	return RingBufferHeaderSize
}

func ReadHeader(ctx context.Context, h *host.Host, bufferPtr uint32) (header *RingBufferHeader, err error) {

	if bufferPtr == 0 {
		return nil, errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return nil, errors.New("host is nil")
	}

	var data []byte
	if data, err = memory.Read(h, bufferPtr, RingBufferHeaderSize); err != nil {
		return nil, fmt.Errorf("failed to read ring buffer header: %w", err)
	}

	if len(data) < RingBufferHeaderSize {
		return nil, errors.New("invalid ring buffer header size")
	}

	header = &RingBufferHeader{
		BufferSize: binary.LittleEndian.Uint32(data[0:4]),
		DataSize:   binary.LittleEndian.Uint32(data[4:8]),
		ReadIndex:  binary.LittleEndian.Uint32(data[8:12]),
		WriteIndex: binary.LittleEndian.Uint32(data[12:16]),
		Closed:     binary.LittleEndian.Uint32(data[16:20]),
	}

	return
}

// ReadIndicesAndClosed: только ReadIndex + WriteIndex + Closed (12 байт) — оптимизация для операций чтения/записи.
func ReadIndicesAndClosed(ctx context.Context, h *host.Host, bufferPtr uint32) (readIdx uint32, writeIdx uint32, closed uint32, err error) {

	if bufferPtr == 0 {
		return 0, 0, 0, errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return 0, 0, 0, errors.New("host is nil")
	}

	var data []byte
	if data, err = memory.Read(h, bufferPtr+8, 12); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read indices and closed flag: %w", err)
	}

	if len(data) < 12 {
		return 0, 0, 0, errors.New("invalid data size for indices and closed flag")
	}

	readIdx = binary.LittleEndian.Uint32(data[0:4])
	writeIdx = binary.LittleEndian.Uint32(data[4:8])
	closed = binary.LittleEndian.Uint32(data[8:12])

	return
}

func WriteHeader(ctx context.Context, h *host.Host, bufferPtr uint32, header *RingBufferHeader) (err error) {

	if bufferPtr == 0 {
		return errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return errors.New("host is nil")
	}

	if header == nil {
		return errors.New("header is nil")
	}

	data := make([]byte, RingBufferHeaderSize)
	binary.LittleEndian.PutUint32(data[0:4], header.BufferSize)
	binary.LittleEndian.PutUint32(data[4:8], header.DataSize)
	binary.LittleEndian.PutUint32(data[8:12], header.ReadIndex)
	binary.LittleEndian.PutUint32(data[12:16], header.WriteIndex)
	binary.LittleEndian.PutUint32(data[16:20], header.Closed)

	if err = memory.Write(h, bufferPtr, data); err != nil {
		return fmt.Errorf("failed to write ring buffer header: %w", err)
	}

	return nil
}

func UpdateReadIndex(ctx context.Context, h *host.Host, bufferPtr uint32, readIndex uint32) (err error) {

	if bufferPtr == 0 {
		return errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return errors.New("host is nil")
	}

	readIndexPtr := bufferPtr + 8 // Смещение для ReadIndex

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, readIndex)

	if err = memory.Write(h, readIndexPtr, data); err != nil {
		return fmt.Errorf("failed to update read index: %w", err)
	}

	return nil
}

func UpdateWriteIndex(ctx context.Context, h *host.Host, bufferPtr uint32, writeIndex uint32) (err error) {

	if bufferPtr == 0 {
		return errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return errors.New("host is nil")
	}

	writeIndexPtr := bufferPtr + 12 // Смещение для WriteIndex

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, writeIndex)

	if err = memory.Write(h, writeIndexPtr, data); err != nil {
		return fmt.Errorf("failed to update write index: %w", err)
	}

	return nil
}

func SetClosed(ctx context.Context, h *host.Host, bufferPtr uint32) (err error) {

	if bufferPtr == 0 {
		return errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return errors.New("host is nil")
	}

	closedPtr := bufferPtr + 16 // Смещение для Closed

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, 1)

	if err = memory.Write(h, closedPtr, data); err != nil {
		return fmt.Errorf("failed to set closed flag: %w", err)
	}

	return nil
}

// AvailableWrite: инвариант — один байт всегда свободен для различения полного/пустого буфера.
func AvailableWrite(readIdx uint32, writeIdx uint32, dataSize uint32) (available uint32) {

	if writeIdx >= readIdx {
		available = dataSize - (writeIdx - readIdx)
		if available > 0 {
			available--
		}
	} else {
		available = readIdx - writeIdx - 1
	}

	return available
}

func AvailableRead(readIdx uint32, writeIdx uint32, dataSize uint32) (available uint32) {

	if writeIdx >= readIdx {
		available = writeIdx - readIdx
	} else {
		available = dataSize - (readIdx - writeIdx)
	}

	return available
}

func WriteToRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32, dataSize uint32, data []byte) (written int, err error) {

	if len(data) == 0 {
		return 0, nil
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if bufferPtr == 0 {
		return 0, errors.New("invalid buffer pointer: zero")
	}

	var readIdx, writeIdx, closed uint32
	if readIdx, writeIdx, closed, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
		return 0, fmt.Errorf("failed to read indices and closed flag: %w", err)
	}

	if closed != 0 {
		return 0, io.EOF
	}

	available := AvailableWrite(readIdx, writeIdx, dataSize)
	if available == 0 {
		return 0, nil
	}

	dataLen := len(data)
	if dataLen < 0 || dataLen > int(^uint32(0)) {
		return 0, errors.New("data length out of range")
	}
	toWrite := uint32(dataLen)
	if toWrite > available {
		toWrite = available
	}

	dataOffset := bufferPtr + RingBufferHeaderSize

	if writeIdx+toWrite <= dataSize {
		dataPtr := dataOffset + writeIdx
		if err = memory.Write(h, dataPtr, data[:toWrite]); err != nil {
			return 0, fmt.Errorf("failed to write data: %w", err)
		}
		writeIdx += toWrite
	} else {
		firstPart := dataSize - writeIdx
		secondPart := toWrite - firstPart

		dataPtr1 := dataOffset + writeIdx
		if err = memory.Write(h, dataPtr1, data[:firstPart]); err != nil {
			return 0, fmt.Errorf("failed to write first part: %w", err)
		}

		dataPtr2 := dataOffset
		if err = memory.Write(h, dataPtr2, data[firstPart:firstPart+secondPart]); err != nil {
			return 0, fmt.Errorf("failed to write second part: %w", err)
		}
		writeIdx = secondPart
	}

	if err = UpdateWriteIndex(ctx, h, bufferPtr, writeIdx); err != nil {
		return 0, fmt.Errorf("failed to update write index: %w", err)
	}

	written = int(toWrite)
	return
}

func ReadFromRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32, dataSize uint32, data []byte) (read int, err error) {

	if len(data) == 0 {
		return 0, nil
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if bufferPtr == 0 {
		return 0, errors.New("invalid buffer pointer: zero")
	}

	var readIdx, writeIdx, closed uint32
	if readIdx, writeIdx, closed, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
		return 0, fmt.Errorf("failed to read indices and closed flag: %w", err)
	}

	available := AvailableRead(readIdx, writeIdx, dataSize)
	if available == 0 {
		if closed != 0 {
			return 0, io.EOF
		}
		return 0, nil
	}

	dataLen := len(data)
	if dataLen < 0 || dataLen > int(^uint32(0)) {
		return 0, errors.New("data length out of range")
	}
	toRead := uint32(dataLen)
	if toRead > available {
		toRead = available
	}

	dataOffset := bufferPtr + RingBufferHeaderSize

	if readIdx+toRead <= dataSize {
		dataPtr := dataOffset + readIdx
		var readData []byte
		if readData, err = memory.Read(h, dataPtr, toRead); err != nil {
			return 0, fmt.Errorf("failed to read data: %w", err)
		}
		copy(data, readData)
		readIdx += toRead
	} else {
		firstPart := dataSize - readIdx
		secondPart := toRead - firstPart

		dataPtr1 := dataOffset + readIdx
		var firstData []byte
		if firstData, err = memory.Read(h, dataPtr1, firstPart); err != nil {
			return 0, fmt.Errorf("failed to read first part: %w", err)
		}
		copy(data, firstData)

		dataPtr2 := dataOffset
		var secondData []byte
		if secondData, err = memory.Read(h, dataPtr2, secondPart); err != nil {
			return 0, fmt.Errorf("failed to read second part: %w", err)
		}
		copy(data[firstPart:], secondData)

		readIdx = secondPart
	}

	if err = UpdateReadIndex(ctx, h, bufferPtr, readIdx); err != nil {
		return 0, fmt.Errorf("failed to update read index: %w", err)
	}

	read = int(toRead)
	return
}

func CreateRingBuffer(ctx context.Context, h *host.Host, size uint32) (bufferPtr uint32, err error) {

	if h == nil {
		return 0, errors.New("host is nil")
	}

	if size < MinRingBufferSize {
		size = MinRingBufferSize
	}
	if size > MaxRingBufferSize {
		size = MaxRingBufferSize
	}

	totalSize := RingBufferHeaderSize + size

	if bufferPtr, err = memory.Allocate(ctx, h, uint64(totalSize)); err != nil {
		return 0, fmt.Errorf("failed to allocate ring buffer: %w", err)
	}

	header := &RingBufferHeader{
		BufferSize: totalSize,
		DataSize:   size,
		ReadIndex:  0,
		WriteIndex: 0,
		Closed:     0,
	}

	if err = WriteHeader(ctx, h, bufferPtr, header); err != nil {
		memory.Free(ctx, h, uint64(bufferPtr))
		return 0, fmt.Errorf("failed to initialize ring buffer header: %w", err)
	}

	return
}

func IsReadBufferEmpty(ctx context.Context, h *host.Host, bufferPtr uint32) (isEmpty bool, err error) {

	if bufferPtr == 0 {
		return false, errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return false, errors.New("host is nil")
	}

	var readIdx, writeIdx, closed uint32
	if readIdx, writeIdx, closed, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
		return false, fmt.Errorf("failed to read indices: %w", err)
	}

	_ = closed
	isEmpty = readIdx == writeIdx
	return
}

// WaitForBufferEmpty: если ReadIndex не меняется в течение stuckTimeout — плагин считаем зависшим; maxWaitTime — защита от бесконечного цикла.
func WaitForBufferEmpty(ctx context.Context, h *host.Host, bufferPtr uint32, checkInterval time.Duration, stuckTimeout time.Duration, maxWaitTime time.Duration) (err error) {

	if bufferPtr == 0 {
		return errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return errors.New("host is nil")
	}

	var lastReadIdx uint32
	var lastReadIdxTime time.Time
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("max wait time exceeded: %v", maxWaitTime)
		}

		var readIdx, writeIdx uint32
		if readIdx, writeIdx, _, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
			return fmt.Errorf("failed to read indices: %w", err)
		}

		if readIdx == writeIdx {
			return nil
		}

		switch {
		case readIdx != lastReadIdx:
			lastReadIdx = readIdx
			lastReadIdxTime = time.Now()
		case !lastReadIdxTime.IsZero():
			if time.Since(lastReadIdxTime) > stuckTimeout {
				return fmt.Errorf("buffer read index not changed for %v, plugin may be stuck", stuckTimeout)
			}
		default:
			lastReadIdx = readIdx
			lastReadIdxTime = time.Now()
		}

		time.Sleep(checkInterval)
	}
}

func DestroyRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32) {

	if bufferPtr == 0 {
		return
	}

	memory.Free(ctx, h, uint64(bufferPtr))
}
