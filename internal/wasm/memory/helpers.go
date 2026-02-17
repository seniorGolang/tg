// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func ValidatePtr(h Host, ptr uint32, size uint32) (err error) {

	if h.GetModule() == nil {
		return errors.New(i18n.Msg("module is not available"))
	}

	mem := h.GetModule().Memory()
	if mem == nil {
		return errors.New(i18n.Msg("memory is not available"))
	}

	memSize := mem.Size()
	if ptr >= memSize {
		return fmt.Errorf(i18n.Msg("invalid memory pointer: ptr=%d >= memSize=%d"), ptr, memSize)
	}

	if size > memSize-ptr {
		return fmt.Errorf(i18n.Msg("invalid memory size: ptr=%d, size=%d, memSize=%d"), ptr, size, memSize)
	}

	return
}
