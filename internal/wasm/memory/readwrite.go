// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func Write(h *host.Host, ptr uint32, data []byte) (err error) {

	if h.Module == nil {
		return errors.New("module is not available")
	}

	mem := h.Module.Memory()
	if mem == nil {
		return errors.New("memory is not available")
	}

	if !mem.Write(ptr, data) {
		return fmt.Errorf("failed to write data to memory at ptr=%d, size=%d", ptr, len(data))
	}

	return nil
}

// Read выполняет валидацию указателя и размера перед чтением из памяти WASM.
func Read(h *host.Host, ptr uint32, size uint32) (data []byte, err error) {

	if h.Module == nil {
		return nil, errors.New("module is not available")
	}

	mem := h.Module.Memory()
	if mem == nil {
		return nil, errors.New("memory is not available")
	}

	if size == 0 {
		return nil, nil
	}

	memSize := mem.Size()
	if ptr >= memSize {
		return nil, fmt.Errorf("invalid memory pointer: ptr=%d >= memSize=%d", ptr, memSize)
	}

	// Проверяем переполнение: ptr + size может переполнить uint32
	if size > memSize-ptr {
		return nil, fmt.Errorf("invalid memory size: ptr=%d, size=%d, memSize=%d", ptr, size, memSize)
	}

	data, ok := mem.Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("failed to read memory: ptr=%d, size=%d, memSize=%d", ptr, size, memSize)
	}

	return
}
