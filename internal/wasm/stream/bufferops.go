// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"time"

	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// writeToRingBufferWithRetry: при полном буфере блокируется до места или отмены контекста.
// readReady - канал для уведомления о готовности к чтению (может быть nil).
func writeToRingBufferWithRetry(ctx context.Context, h memory.Host, bufferPtr uint32, dataSize uint32, data []byte, readReady chan struct{}) (written int, err error) {

	written = 0
	for written < len(data) {
		var w int
		if w, err = WriteToRingBuffer(ctx, h, bufferPtr, dataSize, data[written:]); err != nil {
			return
		}

		if w == 0 {
			// Буфер полон, ждем освобождения места
			if readReady != nil {
				// Используем канал уведомлений
				select {
				case <-ctx.Done():
					return written, ctx.Err()
				case <-readReady:
					// Место освободилось, продолжаем запись
					continue
				}
			}

			// Канала нет, используем короткую паузу для уступки планировщику
			select {
			case <-ctx.Done():
				return written, ctx.Err()
			case <-time.After(time.Millisecond * 1):
				// Короткая пауза для уступки планировщику, затем повторная попытка
				continue
			}
		}

		written += w
	}

	return
}

// readFromRingBufferWithRetry: при пустом буфере блокируется до данных или отмены контекста.
// writeReady - канал для уведомления о готовности к записи (может быть nil).
func readFromRingBufferWithRetry(ctx context.Context, h memory.Host, bufferPtr uint32, dataSize uint32, buffer []byte, writeReady chan struct{}) (n int, err error) {

	for {
		if n, err = ReadFromRingBuffer(ctx, h, bufferPtr, dataSize, buffer); err != nil {
			return
		}

		if n > 0 {
			return
		}

		// Буфер пуст, ждем появления данных
		if writeReady != nil {
			// Используем канал уведомлений
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-writeReady:
				// Данные появились, продолжаем чтение
				continue
			}
		}

		// Канала нет, используем короткую паузу для уступки планировщику
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(time.Millisecond * 1):
			// Короткая пауза для уступки планировщику, затем повторная попытка
			continue
		}
	}
}
