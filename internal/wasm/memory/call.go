// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func CallFunction(ctx context.Context, h Host, funcName string, params ...uint64) (results []uint64, err error) {

	funcPtr := h.GetModule().ExportedFunction(funcName)
	if funcPtr == nil {
		return nil, fmt.Errorf(i18n.Msg("function %s not found in WASM module"), funcName)
	}

	if results, err = funcPtr.Call(ctx, params...); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to call function %s: %w"), funcName, err)
	}

	return
}

func CallFunctionWithRequest[T any](ctx context.Context, h Host, funcName string, request T) (results []uint64, err error) {

	var requestPtr, requestSize uint64
	if requestPtr, requestSize, err = AllocateAndWriteObject(ctx, h, request); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to allocate and write request: %w"), err)
	}

	defer Free(ctx, h, requestPtr)

	return CallFunction(ctx, h, funcName, requestPtr, requestSize)
}
