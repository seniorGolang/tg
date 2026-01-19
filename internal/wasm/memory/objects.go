// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/goccy/go-json"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// AllocateAndWrite выделяет память и записывает в неё данные.
// Возвращает указатель на выделенную память и размер записанных данных.
// Вызывающий код должен освободить память через Free.
func AllocateAndWrite(ctx context.Context, h *host.Host, data []byte) (ptr uint32, size uint32, err error) {

	if len(data) == 0 {
		return 0, 0, nil
	}

	dataLen := len(data)
	if dataLen > int(^uint32(0)) {
		return 0, 0, fmt.Errorf("data size too large for uint32: %d", dataLen)
	}

	size = uint32(dataLen)
	ptr, err = Allocate(ctx, h, uint64(size))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to allocate memory: %w", err)
	}

	if err = Write(h, ptr, data); err != nil {
		// Пытаемся освободить память при ошибке записи
		Free(ctx, h, uint64(ptr))
		return 0, 0, fmt.Errorf("failed to write data: %w", err)
	}

	return
}

// AllocateAndWriteObject маршалит объект типа T в JSON и записывает его в память WASM модуля.
// Возвращает указатель на выделенную память и размер записанных данных в формате uint64 для вызова WASM функций.
// Вызывающий код должен освободить память через Free.
func AllocateAndWriteObject[T any](ctx context.Context, h *host.Host, v T) (ptr uint64, size uint64, err error) {

	var data []byte
	if data, err = json.Marshal(v); err != nil {
		return 0, 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	var ptr32, size32 uint32
	if ptr32, size32, err = AllocateAndWrite(ctx, h, data); err != nil {
		return 0, 0, err
	}

	return uint64(ptr32), uint64(size32), nil
}

// ReadAndUnmarshal валидирует указатель и размер перед чтением JSON из WASM памяти.
// Вызывающий код должен освободить память через Free после использования.
func ReadAndUnmarshal[T any](h *host.Host, ptr uint32, size uint32, v *T) (err error) {

	var data []byte
	if data, err = Read(h, ptr, size); err != nil {
		return fmt.Errorf("failed to read memory: %w", err)
	}

	if err = json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// WriteBytesToPtrSize: стандартный паттерн хоста — результат через ptr/size указатели.
// Память, выделенная для данных, НЕ освобождается автоматически (она будет использоваться WASM модулем).
// Write выполняет валидацию указателей resultPtrPtr и resultSizePtr перед записью.
// Возвращает: 0 - успех, 1 - ошибка.
func WriteBytesToPtrSize(ctx context.Context, h *host.Host, data []byte, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	dataLen := len(data)
	if dataLen == 0 {
		return 1
	}
	if dataLen > int(^uint32(0)) {
		return 1
	}
	dataSize := uint32(dataLen)

	var ptr uint32
	var err error
	if ptr, err = Allocate(ctx, h, uint64(dataSize)); err != nil {
		return 1
	}

	shouldFree := true
	defer func() {
		if shouldFree {
			Free(ctx, h, uint64(ptr))
		}
	}()

	if err = Write(h, ptr, data); err != nil {
		return 1
	}

	ptrBytes := make([]byte, 4)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ptrBytes, ptr)
	binary.LittleEndian.PutUint32(sizeBytes, dataSize)

	if err = Write(h, resultPtrPtr, ptrBytes); err != nil {
		return 1
	}
	if err = Write(h, resultSizePtr, sizeBytes); err != nil {
		return 1
	}

	shouldFree = false
	return 0
}

// WriteObjectToPtrSize маршалит объект типа T в JSON, записывает его в память WASM модуля
// и записывает указатель и размер в указанные места.
// Это стандартный паттерн для экспортируемых функций хоста, которые возвращают результат через указатели.
// Память, выделенная для данных, НЕ освобождается автоматически (она будет использоваться WASM модулем).
// Возвращает: 0 - успех, 1 - ошибка.
func WriteObjectToPtrSize[T any](ctx context.Context, h *host.Host, v T, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	var data []byte
	var err error
	if data, err = json.Marshal(v); err != nil {
		return 1
	}

	return WriteBytesToPtrSize(ctx, h, data, resultPtrPtr, resultSizePtr)
}
