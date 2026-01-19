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
	// RingBufferHeaderSize размер заголовка кольцевого буфера в байтах.
	RingBufferHeaderSize = 20

	// DefaultRingBufferSize размер кольцевого буфера по умолчанию (64KB).
	DefaultRingBufferSize = 64 * 1024

	// MinRingBufferSize минимальный размер кольцевого буфера (4KB).
	MinRingBufferSize = 4 * 1024

	// MaxRingBufferSize максимальный размер кольцевого буфера (1MB).
	MaxRingBufferSize = 1024 * 1024
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

// ReadHeader только при создании буфера; для чтения/записи используйте ReadIndicesAndClosed.
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

	// Читаем только 12 байт: ReadIndex (8-11) + WriteIndex (12-15) + Closed (16-19)
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

// UpdateReadIndex атомарно обновляет индекс чтения в WASM памяти.
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

// UpdateWriteIndex атомарно обновляет индекс записи в WASM памяти.
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

// AvailableWrite вычисляет доступное место для записи в кольцевой буфер.
// Инвариант: один байт всегда остается свободным для различения полного/пустого буфера.
func AvailableWrite(readIdx uint32, writeIdx uint32, dataSize uint32) (available uint32) {

	if writeIdx >= readIdx {
		// Обычный случай: writeIdx >= readIdx
		available = dataSize - (writeIdx - readIdx)
		if available > 0 {
			available-- // Оставляем один байт для различения полного/пустого буфера
		}
	} else {
		// Wrap-around: writeIdx < readIdx (запись обогнала чтение)
		available = readIdx - writeIdx - 1
	}

	return available
}

// AvailableRead вычисляет доступные данные для чтения из кольцевого буфера.
func AvailableRead(readIdx uint32, writeIdx uint32, dataSize uint32) (available uint32) {

	if writeIdx >= readIdx {
		// Обычный случай: writeIdx >= readIdx
		available = writeIdx - readIdx
	} else {
		// Wrap-around: writeIdx < readIdx
		available = dataSize - (readIdx - writeIdx)
	}

	return available
}

// WriteToRingBuffer оптимизировано для минимизации операций с памятью.
// dataSize - кэшированный размер области данных (константа для буфера).
// memory.Write выполняет валидацию указателей перед записью.
func WriteToRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32, dataSize uint32, data []byte) (written int, err error) {

	if len(data) == 0 {
		return 0, nil
	}

	// Проверяем контекст перед началом операции
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if bufferPtr == 0 {
		return 0, errors.New("invalid buffer pointer: zero")
	}

	// Читаем только индексы и флаг закрытия (12 байт вместо 20)
	var readIdx, writeIdx, closed uint32
	if readIdx, writeIdx, closed, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
		return 0, fmt.Errorf("failed to read indices and closed flag: %w", err)
	}

	if closed != 0 {
		return 0, io.EOF
	}

	// Вычисляем доступное место, используя кэшированный dataSize
	available := AvailableWrite(readIdx, writeIdx, dataSize)
	if available == 0 {
		return 0, nil // Буфер полон
	}

	// Ограничиваем размер данных доступным местом
	// Проверяем переполнение при конвертации int -> uint32
	dataLen := len(data)
	if dataLen < 0 || dataLen > int(^uint32(0)) {
		return 0, errors.New("data length out of range")
	}
	toWrite := uint32(dataLen)
	if toWrite > available {
		toWrite = available
	}

	dataOffset := bufferPtr + RingBufferHeaderSize

	// Записываем данные с учетом wrap-around
	if writeIdx+toWrite <= dataSize {
		// Обычный случай: данные не пересекают границу буфера
		dataPtr := dataOffset + writeIdx
		if err = memory.Write(h, dataPtr, data[:toWrite]); err != nil {
			return 0, fmt.Errorf("failed to write data: %w", err)
		}
		writeIdx += toWrite
	} else {
		// Wrap-around: данные пересекают границу буфера
		firstPart := dataSize - writeIdx
		secondPart := toWrite - firstPart

		// Записываем первую часть
		dataPtr1 := dataOffset + writeIdx
		if err = memory.Write(h, dataPtr1, data[:firstPart]); err != nil {
			return 0, fmt.Errorf("failed to write first part: %w", err)
		}

		// Записываем вторую часть
		dataPtr2 := dataOffset
		if err = memory.Write(h, dataPtr2, data[firstPart:firstPart+secondPart]); err != nil {
			return 0, fmt.Errorf("failed to write second part: %w", err)
		}
		writeIdx = secondPart
	}

	// Атомарно обновляем WriteIndex
	if err = UpdateWriteIndex(ctx, h, bufferPtr, writeIdx); err != nil {
		return 0, fmt.Errorf("failed to update write index: %w", err)
	}

	written = int(toWrite)
	return
}

// ReadFromRingBuffer оптимизировано для минимизации копирований и аллокаций.
// dataSize - кэшированный размер области данных (константа для буфера).
// memory.Read выполняет валидацию указателей перед чтением.
func ReadFromRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32, dataSize uint32, data []byte) (read int, err error) {

	if len(data) == 0 {
		return 0, nil
	}

	// Проверяем контекст перед началом операции
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if bufferPtr == 0 {
		return 0, errors.New("invalid buffer pointer: zero")
	}

	// Читаем только индексы и флаг закрытия (12 байт вместо 20)
	var readIdx, writeIdx, closed uint32
	if readIdx, writeIdx, closed, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
		return 0, fmt.Errorf("failed to read indices and closed flag: %w", err)
	}

	// Вычисляем доступные данные, используя кэшированный dataSize
	available := AvailableRead(readIdx, writeIdx, dataSize)
	if available == 0 {
		if closed != 0 {
			return 0, io.EOF
		}
		return 0, nil // Буфер пуст
	}

	// Ограничиваем размер данных доступными данными
	// Проверяем переполнение при конвертации int -> uint32
	dataLen := len(data)
	if dataLen < 0 || dataLen > int(^uint32(0)) {
		return 0, errors.New("data length out of range")
	}
	toRead := uint32(dataLen)
	if toRead > available {
		toRead = available
	}

	dataOffset := bufferPtr + RingBufferHeaderSize

	// Читаем данные с учетом wrap-around, минимизируя копирования
	if readIdx+toRead <= dataSize {
		// Обычный случай: данные не пересекают границу буфера
		dataPtr := dataOffset + readIdx
		var readData []byte
		if readData, err = memory.Read(h, dataPtr, toRead); err != nil {
			return 0, fmt.Errorf("failed to read data: %w", err)
		}
		// Копируем напрямую в выходной буфер
		copy(data, readData)
		readIdx += toRead
	} else {
		// Wrap-around: данные пересекают границу буфера
		// Читаем напрямую в выходной буфер двумя частями
		firstPart := dataSize - readIdx
		secondPart := toRead - firstPart

		// Читаем первую часть напрямую в начало выходного буфера
		dataPtr1 := dataOffset + readIdx
		var firstData []byte
		if firstData, err = memory.Read(h, dataPtr1, firstPart); err != nil {
			return 0, fmt.Errorf("failed to read first part: %w", err)
		}
		copy(data, firstData)

		// Читаем вторую часть напрямую в продолжение выходного буфера
		dataPtr2 := dataOffset
		var secondData []byte
		if secondData, err = memory.Read(h, dataPtr2, secondPart); err != nil {
			return 0, fmt.Errorf("failed to read second part: %w", err)
		}
		copy(data[firstPart:], secondData)

		readIdx = secondPart
	}

	// Атомарно обновляем ReadIndex
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

	// Нормализуем размер буфера
	if size < MinRingBufferSize {
		size = MinRingBufferSize
	}
	if size > MaxRingBufferSize {
		size = MaxRingBufferSize
	}

	// Вычисляем общий размер (заголовок + данные)
	totalSize := RingBufferHeaderSize + size

	// Выделяем память в WASM
	if bufferPtr, err = memory.Allocate(ctx, h, uint64(totalSize)); err != nil {
		return 0, fmt.Errorf("failed to allocate ring buffer: %w", err)
	}

	// Инициализируем заголовок
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

// IsReadBufferEmpty: true, если ReadIndex == WriteIndex; используется для синхронизации хост/плагин.
func IsReadBufferEmpty(ctx context.Context, h *host.Host, bufferPtr uint32) (isEmpty bool, err error) {

	if bufferPtr == 0 {
		return false, errors.New("invalid buffer pointer: zero")
	}

	if h == nil {
		return false, errors.New("host is nil")
	}

	// Читаем только индексы (8 байт) вместо всего заголовка
	var readIdx, writeIdx, closed uint32
	if readIdx, writeIdx, closed, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
		return false, fmt.Errorf("failed to read indices: %w", err)
	}

	// Буфер пуст, если ReadIndex == WriteIndex
	// closed не используется, но уже прочитан в ReadIndicesAndClosed
	_ = closed
	isEmpty = readIdx == writeIdx
	return
}

// WaitForBufferEmpty ждёт, пока буфер не станет пустым (ReadIndex == WriteIndex).
// Отслеживает изменение ReadIndex для определения зависшего плагина.
// Если ReadIndex не меняется в течение stuckTimeout, значит плагин завис и функция завершается.
// Если ReadIndex меняется, значит плагин читает данные, и функция ждёт дольше.
// checkInterval - интервал проверки состояния буфера.
// stuckTimeout - таймаут для определения зависшего плагина (если ReadIndex не меняется).
// maxWaitTime - максимальное время ожидания (защита от бесконечного цикла).
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
		// Проверяем контекст
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Проверяем максимальное время ожидания
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("max wait time exceeded: %v", maxWaitTime)
		}

		// Читаем индексы напрямую для отслеживания изменения ReadIndex
		var readIdx, writeIdx uint32
		if readIdx, writeIdx, _, err = ReadIndicesAndClosed(ctx, h, bufferPtr); err != nil {
			return fmt.Errorf("failed to read indices: %w", err)
		}

		// Если буфер пуст (ReadIndex == WriteIndex), все данные прочитаны - выходим
		if readIdx == writeIdx {
			return nil
		}

		// Отслеживаем изменение ReadIndex
		switch {
		case readIdx != lastReadIdx:
			// ReadIndex изменился - плагин читает данные, сбрасываем таймер
			lastReadIdx = readIdx
			lastReadIdxTime = time.Now()
		case !lastReadIdxTime.IsZero():
			// ReadIndex не меняется - проверяем, не завис ли плагин
			if time.Since(lastReadIdxTime) > stuckTimeout {
				// ReadIndex не менялся в течение stuckTimeout - плагин завис
				return fmt.Errorf("buffer read index not changed for %v, plugin may be stuck", stuckTimeout)
			}
		default:
			// Первая итерация - инициализируем
			lastReadIdx = readIdx
			lastReadIdxTime = time.Now()
		}

		// Ждём перед следующей проверкой
		time.Sleep(checkInterval)
	}
}

// DestroyRingBuffer освобождает память кольцевого буфера.
func DestroyRingBuffer(ctx context.Context, h *host.Host, bufferPtr uint32) {

	if bufferPtr == 0 {
		return
	}

	memory.Free(ctx, h, uint64(bufferPtr))
}
