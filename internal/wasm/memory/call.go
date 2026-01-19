// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// CallFunction вызывает экспортированную WASM функцию с переданными параметрами.
// Выполняет проверку наличия функции и обработку результатов.
func CallFunction(ctx context.Context, h *host.Host, funcName string, params ...uint64) (results []uint64, err error) {

	funcPtr := h.Module.ExportedFunction(funcName)
	if funcPtr == nil {
		return nil, fmt.Errorf("function %s not found in WASM module", funcName)
	}

	if results, err = funcPtr.Call(ctx, params...); err != nil {
		return nil, fmt.Errorf("failed to call function %s: %w", funcName, err)
	}

	return
}

// CallFunctionWithRequest вызывает WASM функцию, передавая запрос в памяти.
// Выделяет память, записывает запрос, вызывает функцию и автоматически освобождает память.
// Возвращает результат вызова функции.
func CallFunctionWithRequest[T any](ctx context.Context, h *host.Host, funcName string, request T) (results []uint64, err error) {

	var requestPtr, requestSize uint64
	if requestPtr, requestSize, err = AllocateAndWriteObject(ctx, h, request); err != nil {
		return nil, fmt.Errorf("failed to allocate and write request: %w", err)
	}

	defer Free(ctx, h, requestPtr)

	return CallFunction(ctx, h, funcName, requestPtr, requestSize)
}

// CallFunctionWithResult вызывает WASM функцию и обрабатывает результат выполнения.
// Парсит результат, читает данные из памяти и возвращает распарсенный ответ.
func CallFunctionWithResult[TRequest, TResult any](ctx context.Context, h *host.Host, funcName string, request TRequest) (resp *TResult, err error) {

	var results []uint64
	if results, err = CallFunctionWithRequest(ctx, h, funcName, request); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New("function returned no results")
	}

	return ReadExecuteResponse[TResult](ctx, h, results[0])
}
