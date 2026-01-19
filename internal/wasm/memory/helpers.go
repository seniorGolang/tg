// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// ReadUint32 читает uint32 из памяти WASM модуля (little-endian, 4 байта).
func ReadUint32(h *host.Host, ptr uint32) (value uint32, err error) {

	var data []byte
	if data, err = Read(h, ptr, 4); err != nil {
		return 0, err
	}

	if len(data) != 4 {
		return 0, fmt.Errorf("invalid data size for uint32: expected 4, got %d", len(data))
	}

	value = binary.LittleEndian.Uint32(data)
	return
}

func ValidatePtr(h *host.Host, ptr uint32, size uint32) (err error) {

	if h.Module == nil {
		return errors.New("module is not available")
	}

	mem := h.Module.Memory()
	if mem == nil {
		return errors.New("memory is not available")
	}

	memSize := mem.Size()
	if ptr >= memSize {
		return fmt.Errorf("invalid memory pointer: ptr=%d >= memSize=%d", ptr, memSize)
	}

	// Проверяем переполнение: ptr + size может переполнить uint32
	if size > memSize-ptr {
		return fmt.Errorf("invalid memory size: ptr=%d, size=%d, memSize=%d", ptr, size, memSize)
	}

	return nil
}

// ReadPtrSize: little-endian, 8 байт (ptr + size); ptr указывает на начало структуры.
func ReadPtrSize(h *host.Host, ptr uint32) (dataPtr uint32, dataSize uint32, err error) {

	var ptrBytes []byte
	if ptrBytes, err = Read(h, ptr, 4); err != nil {
		return 0, 0, fmt.Errorf("failed to read pointer: %w", err)
	}

	// Проверяем переполнение при вычислении ptr+4
	ptrOffset := uint64(ptr) + 4
	if ptrOffset > uint64(^uint32(0)) {
		return 0, 0, errors.New("ptr+4 too large for uint32")
	}

	var sizeBytes []byte
	if sizeBytes, err = Read(h, uint32(ptrOffset), 4); err != nil {
		return 0, 0, fmt.Errorf("failed to read size: %w", err)
	}

	dataPtr = binary.LittleEndian.Uint32(ptrBytes)
	dataSize = binary.LittleEndian.Uint32(sizeBytes)

	return
}

func ValidatePtrAndSize(ptr uint32, size uint32) (err error) {

	if size == 0 {
		return errors.New("size is zero")
	}

	if ptr == 0 {
		return errors.New("pointer is zero")
	}

	return nil
}

// ReadAndUnmarshalValidated: проверка на нулевые значения и валидация указателя перед чтением.
// Автоматически освобождает память после использования.
func ReadAndUnmarshalValidated[T any](ctx context.Context, h *host.Host, ptr uint32, size uint32, v *T) (err error) {

	if err := ValidatePtrAndSize(ptr, size); err != nil {
		return err
	}

	// Валидируем указатель перед чтением
	if err := ValidatePtr(h, ptr, size); err != nil {
		return fmt.Errorf("invalid memory pointer: %w", err)
	}

	defer Free(ctx, h, uint64(ptr))

	// Читаем и анмаршалим данные используя менеджер памяти
	if err := ReadAndUnmarshal(h, ptr, size, v); err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	return nil
}
