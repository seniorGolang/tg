// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// ParseExecuteResult парсит результат выполнения плагина из uint64.
// Формат: верхние 32 бита - указатель, нижние 32 бита - размер (31-й бит - флаг ошибки).
func ParseExecuteResult(result uint64) (ptr uint32, size uint32, hasError bool, err error) {

	upper32 := result >> 32
	if upper32 > math.MaxUint32 {
		return 0, 0, false, fmt.Errorf(i18n.Msg("result pointer overflow: %d"), upper32)
	}

	ptr = uint32(upper32)
	hasError = (result & (1 << 31)) != 0
	// Младшие 32 бита: бит 31 — флаг ошибки, биты 0–30 — размер. Маска 0x7FFFFFFF выделяет только размер.
	lower32 := uint32(result & 0xFFFFFFFF) //nolint:gosec // безопасное преобразование через маску
	size = lower32 & 0x7FFFFFFF

	return
}

// ReadAndUnmarshalWithErrorLogging валидирует указатель и логирует ошибки декодирования.
// Автоматически освобождает память после использования.
func ReadAndUnmarshalWithErrorLogging[T any](ctx context.Context, h *host.Host, ptr uint32, size uint32, v *T) (err error) {

	if size == 0 {
		return nil
	}

	// Валидируем указатель перед чтением
	if err := ValidatePtr(h, ptr, size); err != nil {
		return fmt.Errorf(i18n.Msg("invalid memory pointer: %w"), err)
	}

	defer Free(ctx, h, uint64(ptr))

	// Читаем и анмаршалим ответ используя менеджер памяти
	if err := ReadAndUnmarshal(h, ptr, size, v); err != nil {
		// Логируем ошибку декодирования для отладки
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

// ReadExecuteResponse обрабатывает результат выполнения плагина: парсит результат, читает данные из памяти
// и возвращает распарсенный ответ. Автоматически обрабатывает все ошибки и освобождает память.
func ReadExecuteResponse[T any](ctx context.Context, h *host.Host, result uint64) (resp *T, err error) {

	var ptr, size uint32
	if ptr, size, _, err = ParseExecuteResult(result); err != nil {
		return nil, err
	}

	if size == 0 {
		return nil, nil
	}

	var response T
	if err = ReadAndUnmarshalWithErrorLogging(ctx, h, ptr, size, &response); err != nil {
		return nil, err
	}

	resp = &response
	return
}
