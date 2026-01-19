// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package imports

import (
	"context"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// callWithResult вызывает WASM функцию и обрабатывает результат.
// Возвращает распарсенный результат или ошибку.
func callWithResult[T any](ctx context.Context, h *host.Host, channel *host.CallChannel, functionName string, requestData []byte) (resp *T, err error) {

	resultChan := channel.Call(functionName, requestData)

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", i18n.Msg("context cancelled"), ctx.Err())

	case result := <-resultChan:
		if result.Error != nil {
			return nil, result.Error
		}

		if result.Result == 0 {
			return nil, nil
		}

		return memory.ReadExecuteResponse[T](ctx, h, result.Result)
	}
}
