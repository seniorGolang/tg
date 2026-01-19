// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"io"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// StreamReader реализует io.Reader для чтения данных из потока через кольцевой буфер.
// Хост читает данные из io.Reader и записывает их в кольцевой буфер в WASM памяти.
type StreamReader struct {
	streamID uint32
	registry *StreamRegistry
	host     *host.Host
	ctx      context.Context
}

func NewStreamReader(ctx context.Context, streamID uint32, registry *StreamRegistry, h *host.Host) (reader *StreamReader) {

	return &StreamReader{
		streamID: streamID,
		registry: registry,
		host:     h,
		ctx:      ctx,
	}
}

// Read: хост читает из buffer и пишет в кольцевой буфер; WASM читает из буфера напрямую.
func (r *StreamReader) Read(buffer []byte) (n int, err error) {

	state, ok := r.registry.GetStreamStateDirect(r.streamID)
	if !ok || state == nil {
		return 0, io.ErrClosedPipe
	}

	state.Mu.RLock()
	closed := state.Closed
	reader := state.Reader
	readBuffer := state.ReadBuffer
	state.Mu.RUnlock()

	if closed {
		return 0, io.EOF
	}

	if reader == nil {
		return 0, io.ErrClosedPipe
	}

	// Используем временный буфер для чтения из Reader
	if len(readBuffer) < len(buffer) {
		readBuffer = make([]byte, len(buffer))
	} else {
		readBuffer = readBuffer[:len(buffer)]
	}

	// Читаем данные из io.Reader
	var readN int
	if readN, err = reader.Read(readBuffer); err != nil && err != io.EOF {
		return 0, err
	}

	if readN == 0 {
		if err == io.EOF {
			// Закрываем поток при EOF
			r.registry.CloseStream(r.ctx, r.host, r.streamID)
			return 0, io.EOF
		}
		return 0, nil
	}

	// Записываем данные в кольцевой буфер для чтения (хост → WASM)
	// Блокируем до полной записи всех данных, если буфер полон
	// В Read нет канала уведомлений, поэтому передаем nil
	written, err := writeToRingBufferWithRetry(r.ctx, r.host, state.ReadBufferPtr, state.ReadBufferDataSize, readBuffer[:readN], nil)
	if err != nil {
		return 0, err
	}

	n = written
	return
}

// StartReader запускает горутину для непрерывного чтения данных из Reader
// и записи их в кольцевой буфер.
func (r *StreamReader) StartReader() {

	// Получаем state и увеличиваем счетчик WaitGroup перед запуском горутины
	state, ok := r.registry.GetStreamStateDirect(r.streamID)
	if !ok || state == nil {
		return
	}
	state.readerDone.Add(1)

	go func() {
		// Уменьшаем счетчик WaitGroup при завершении горутины
		defer state.readerDone.Done()

		buffer := make([]byte, 4096)
		for {
			// Проверяем контекст
			select {
			case <-r.ctx.Done():
				return
			default:
			}

			state, ok := r.registry.GetStreamStateDirect(r.streamID)
			if !ok || state == nil {
				return
			}

			state.Mu.RLock()
			closed := state.Closed
			reader := state.Reader
			state.Mu.RUnlock()

			if closed || reader == nil {
				return
			}

			// Читаем данные из Reader
			// ВАЖНО: pipe.Read() блокирует до появления данных или EOF
			// Для коротких команд (echo) данные могут быть доступны сразу
			n, err := reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					// Команда завершилась, данных больше не будет
					// НО: возможно, мы уже прочитали данные, но еще не записали их в буфер
					// Проверяем, есть ли данные для записи
					if n > 0 {
						// Есть данные, записываем их перед закрытием
						// В StartReader нет канала уведомлений, поэтому передаем nil
						written, writeErr := writeToRingBufferWithRetry(r.ctx, r.host, state.ReadBufferPtr, state.ReadBufferDataSize, buffer[:n], nil)
						if writeErr != nil {
							if writeErr == io.EOF {
								r.registry.CloseStream(r.ctx, r.host, r.streamID)
							}
							return
						}
						// Проверяем, что все данные записаны
						if written != n {
							// Не все данные записаны - это ошибка
							return
						}
					}

					// Помечаем поток как закрытый для чтения
					state.Mu.Lock()
					state.Reader = nil
					state.Mu.Unlock()

					// Выходим из горутины - больше данных не будет
					// readerDone.Done() будет вызван в defer
					return
				}

				// Другие ошибки
				return
			}

			if n == 0 {
				continue
			}

			// Записываем данные в кольцевой буфер для чтения (хост → WASM)
			// Блокируем до появления места в буфере, если буфер полон
			// В StartReader нет канала уведомлений, поэтому передаем nil
			written, writeErr := writeToRingBufferWithRetry(r.ctx, r.host, state.ReadBufferPtr, state.ReadBufferDataSize, buffer[:n], nil)
			if writeErr != nil {
				if writeErr == io.EOF {
					r.registry.CloseStream(r.ctx, r.host, r.streamID)
				}
				return
			}
			// Проверяем, что все данные записаны
			if written != n {
				// Не все данные записаны - это ошибка
				return
			}
		}
	}()
}
