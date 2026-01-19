// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"io"
	"time"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// StreamWriter реализует io.Writer для записи данных в поток через кольцевой буфер.
// StreamWriter: WASM пишет в кольцевой буфер, хост читает из буфера и пишет в io.Writer.
type StreamWriter struct {
	streamID uint32
	registry *StreamRegistry
	host     *host.Host
	ctx      context.Context
}

func NewStreamWriter(ctx context.Context, streamID uint32, registry *StreamRegistry, h *host.Host) (writer *StreamWriter) {

	return &StreamWriter{
		streamID: streamID,
		registry: registry,
		host:     h,
		ctx:      ctx,
	}
}

// Write: хост читает из кольцевого буфера (куда пишет WASM) и записывает в io.Writer.
func (w *StreamWriter) Write(buffer []byte) (n int, err error) {

	state, ok := w.registry.GetStreamStateDirect(w.streamID)
	if !ok || state == nil {
		return 0, io.ErrClosedPipe
	}

	state.Mu.RLock()
	closed := state.Closed
	writer := state.Writer
	writeBuffer := state.WriteBuffer
	state.Mu.RUnlock()

	if closed {
		return 0, io.ErrClosedPipe
	}

	if writer == nil {
		return 0, io.ErrClosedPipe
	}

	// Используем временный буфер для чтения из кольцевого буфера
	if len(writeBuffer) < len(buffer) {
		writeBuffer = make([]byte, len(buffer))
	} else {
		writeBuffer = writeBuffer[:len(buffer)]
	}

	// Читаем данные из кольцевого буфера для записи (WASM → хост)
	var readN int
	if readN, err = ReadFromRingBuffer(w.ctx, w.host, state.WriteBufferPtr, state.WriteBufferDataSize, writeBuffer); err != nil {
		if err == io.EOF {
			w.registry.CloseStream(w.ctx, w.host, w.streamID)
		}
		return 0, err
	}

	if readN == 0 {
		return 0, nil
	}

	// Записываем данные в io.Writer
	// Блокируем до полной записи всех данных
	totalWritten := 0
	for totalWritten < readN {
		var writtenBytes int
		var writeErr error
		if writtenBytes, writeErr = writer.Write(writeBuffer[totalWritten:readN]); writeErr != nil {
			return 0, writeErr
		}

		if writtenBytes == 0 {
			// Writer не может записать данные, возвращаем то, что успели записать
			break
		}

		totalWritten += writtenBytes
	}

	n = totalWritten
	return
}

// StartWriter запускает горутину для непрерывного чтения данных из кольцевого буфера
// и записи их в Writer.
func (w *StreamWriter) StartWriter() {

	go func() {
		buffer := make([]byte, 4096)
		for {
			// Проверяем контекст
			select {
			case <-w.ctx.Done():
				return
			default:
			}

			state, ok := w.registry.GetStreamStateDirect(w.streamID)
			if !ok || state == nil {
				return
			}

			state.Mu.RLock()
			closed := state.Closed
			writer := state.Writer
			state.Mu.RUnlock()

			if closed || writer == nil {
				return
			}

			// Читаем данные из кольцевого буфера для записи (WASM → хост)
			// В StartWriter нет канала уведомлений, поэтому передаем nil
			n, err := readFromRingBufferWithRetry(w.ctx, w.host, state.WriteBufferPtr, state.WriteBufferDataSize, buffer, nil)
			if err != nil {
				if err == io.EOF {
					w.registry.CloseStream(w.ctx, w.host, w.streamID)
				}
				return
			}

			// Записываем данные в Writer
			// Блокируем до полной записи всех данных, если Writer возвращает частичную запись
			totalWritten := 0
			for totalWritten < n {
				select {
				case <-w.ctx.Done():
					return
				default:
				}

				var writtenBytes int
				var writeErr error
				if writtenBytes, writeErr = writer.Write(buffer[totalWritten:n]); writeErr != nil {
					return
				}

				if writtenBytes == 0 {
					// Writer не может записать данные, это нормально для некоторых типов Writer
					// Продолжаем попытку записи в следующей итерации
					// Используем короткую паузу для уступки планировщику
					select {
					case <-w.ctx.Done():
						return
					case <-time.After(time.Millisecond * 1):
						// Короткая пауза для уступки планировщику, затем повторная попытка
						continue
					}
				}

				totalWritten += writtenBytes
			}
		}
	}()
}
