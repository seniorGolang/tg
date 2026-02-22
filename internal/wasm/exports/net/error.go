// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// writeError: формат uint64 — верхние 32 бита указатель на строку, нижние 32 длина + флаг ошибки (31-й бит).
func writeError(ctx context.Context, h *host.Host, err error) (result uint64) {

	if err == nil {
		return 0
	}

	errStr := err.Error()
	errBytes := []byte(errStr)

	var errPtr uint32
	var allocErr error
	if errPtr, allocErr = memory.Allocate(ctx, h, uint64(len(errBytes))); allocErr != nil {
		// При ошибке аллокации гость не сможет прочитать строку; возвращаем 0 как признак ошибки.
		return 0
	}

	var writeErr error
	if writeErr = memory.Write(h, errPtr, errBytes); writeErr != nil {
		memory.Free(ctx, h, uint64(errPtr))
		return 0
	}

	// Формат результата: верхние 32 бита — указатель, нижние 32 — длина с флагом ошибки (31-й бит).
	errLen := len(errBytes)
	if errLen > int(^uint32(0)>>1) {
		errLen = int(^uint32(0) >> 1)
	}
	length := uint32(errLen) | (uint32(1) << 31) //nolint:gosec // проверка на переполнение выполнена выше
	return (uint64(errPtr) << 32) | uint64(length)
}
