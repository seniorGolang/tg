// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// Allocate выделяет память в WASM модуле через malloc.
// Возвращает указатель на выделенную память или ошибку.
func Allocate(ctx context.Context, h *host.Host, size uint64) (ptr uint32, err error) {

	if h.Malloc == nil {
		return 0, errors.New("malloc function is not available")
	}

	if size == 0 {
		return 0, nil
	}

	var results []uint64
	if results, err = h.Malloc.Call(ctx, size); err != nil {
		return 0, fmt.Errorf("failed to allocate memory: %w", err)
	}

	if len(results) == 0 {
		return 0, errors.New("malloc returned no results")
	}

	allocatedPtr := results[0]
	if allocatedPtr > uint64(^uint32(0)) {
		return 0, errors.New("allocated pointer too large for uint32")
	}

	ptr = uint32(allocatedPtr)
	return
}

// Free освобождает память в WASM модуле через free.
func Free(ctx context.Context, h *host.Host, ptr uint64) {

	if h.Free == nil {
		return
	}

	if ptr == 0 {
		return
	}

	_, _ = h.Free.Call(ctx, ptr)
}
