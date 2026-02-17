// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func Write(h Host, ptr uint32, data []byte) (err error) {

	if h.GetModule() == nil {
		return errors.New(i18n.Msg("module is not available"))
	}

	mem := h.GetModule().Memory()
	if mem == nil {
		return errors.New(i18n.Msg("memory is not available"))
	}

	if !mem.Write(ptr, data) {
		return fmt.Errorf(i18n.Msg("failed to write data to memory at ptr=%d, size=%d"), ptr, len(data))
	}

	return nil
}

func Read(h Host, ptr uint32, size uint32) (data []byte, err error) {

	if h.GetModule() == nil {
		return nil, errors.New(i18n.Msg("module is not available"))
	}

	mem := h.GetModule().Memory()
	if mem == nil {
		return nil, errors.New(i18n.Msg("memory is not available"))
	}

	if size == 0 {
		return nil, nil
	}

	memSize := mem.Size()
	if ptr >= memSize {
		return nil, fmt.Errorf(i18n.Msg("invalid memory pointer: ptr=%d >= memSize=%d"), ptr, memSize)
	}

	if size > memSize-ptr {
		return nil, fmt.Errorf(i18n.Msg("invalid memory size: ptr=%d, size=%d, memSize=%d"), ptr, size, memSize)
	}

	data, ok := mem.Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf(i18n.Msg("failed to read memory: ptr=%d, size=%d, memSize=%d"), ptr, size, memSize)
	}

	return
}
