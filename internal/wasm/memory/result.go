// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func ParseExecuteResult(result uint64) (ptr uint32, size uint32, hasError bool, err error) {

	upper32 := result >> 32
	if upper32 > math.MaxUint32 {
		return 0, 0, false, fmt.Errorf(i18n.Msg("result pointer overflow: %d"), upper32)
	}

	lower32 := uint32(result & 0xFFFFFFFF) //nolint:gosec // маска для размера и флага ошибки
	return uint32(upper32), lower32 & 0x7FFFFFFF, (result & (1 << 31)) != 0, nil
}

func ReadAndUnmarshalWithErrorLogging[T any](ctx context.Context, h Host, ptr uint32, size uint32, v *T) (err error) {

	if size == 0 {
		return nil
	}

	if err = ValidatePtr(h, ptr, size); err != nil {
		return fmt.Errorf(i18n.Msg("invalid memory pointer: %w"), err)
	}

	defer Free(ctx, h, uint64(ptr))

	if err = ReadAndUnmarshal(h, ptr, size, v); err != nil {
		responseBytes, readErr := Read(h, ptr, size)
		if readErr == nil {
			firstBytes := responseBytes
			if len(firstBytes) > 100 {
				firstBytes = firstBytes[:100]
			}
			slog.Error(i18n.Msg("host.Execute: decoding error"), "error", err, "response_size", len(responseBytes), "first_100_bytes", string(firstBytes))
		}
		return fmt.Errorf(i18n.Msg("failed to decode response: %w"), err)
	}

	return nil
}

func ReadExecuteResponse[T any](ctx context.Context, h Host, result uint64) (resp *T, err error) {

	var ptr, size uint32
	if ptr, size, _, err = ParseExecuteResult(result); err != nil {
		return
	}

	if size == 0 {
		return
	}

	var response T
	if err = ReadAndUnmarshalWithErrorLogging(ctx, h, ptr, size, &response); err != nil {
		return
	}

	return &response, nil
}
