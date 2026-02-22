// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"io"

	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// StreamReader реализует io.Reader для чтения данных из потока через кольцевой буфер.
// Хост читает данные из io.Reader и записывает их в кольцевой буфер в WASM памяти.
type Reader struct {
	streamID uint32
	registry registry
	host     memory.Host
	ctx      context.Context
}

func NewStreamReader(ctx context.Context, streamID uint32, registry registry, h memory.Host) (reader *Reader) {

	return &Reader{
		host:     h,
		ctx:      ctx,
		streamID: streamID,
		registry: registry,
	}
}

// Read: хост читает из buffer и пишет в кольцевой буфер; WASM читает из буфера напрямую.
func (r *Reader) Read(buffer []byte) (n int, err error) {

	state, ok := r.registry.GetStreamState(r.streamID)
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

	if len(readBuffer) < len(buffer) {
		readBuffer = make([]byte, len(buffer))
	} else {
		readBuffer = readBuffer[:len(buffer)]
	}

	var readN int
	if readN, err = reader.Read(readBuffer); err != nil && err != io.EOF {
		return 0, err
	}

	if readN == 0 {
		if err == io.EOF {
			r.registry.CloseStream(r.ctx, r.host, r.streamID)
			return 0, io.EOF
		}
		return 0, nil
	}

	// Блокируем до полной записи всех данных, если буфер полон. В Read нет канала уведомлений, поэтому передаем nil.
	var written int
	if written, err = writeToRingBufferWithRetry(r.ctx, r.host, state.ReadBufferPtr, state.ReadBufferDataSize, readBuffer[:readN], nil); err != nil {
		return 0, err
	}

	return written, nil
}

// StartReader запускает горутину для непрерывного чтения данных из Reader
// и записи их в кольцевой буфер.
func (r *Reader) StartReader() {

	state, ok := r.registry.GetStreamState(r.streamID)
	if !ok || state == nil {
		return
	}
	state.readerDone.Add(1)

	go func() {
		defer state.readerDone.Done()

		buffer := make([]byte, 4096)
		for {
			select {
			case <-r.ctx.Done():
				return
			default:
			}

			state, ok = r.registry.GetStreamState(r.streamID)
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

			// pipe.Read блокирует до данных или EOF; при EOF возможно n>0 — данные уже прочитаны, но ещё не записаны в буфер.
			var n int
			var err error
			if n, err = reader.Read(buffer); err != nil {
				if err == io.EOF {
					if n > 0 {
						written, writeErr := writeToRingBufferWithRetry(r.ctx, r.host, state.ReadBufferPtr, state.ReadBufferDataSize, buffer[:n], nil)
						if writeErr != nil {
							if writeErr == io.EOF {
								r.registry.CloseStream(r.ctx, r.host, r.streamID)
							}
							return
						}
						if written != n {
							return
						}
					}

					state.Mu.Lock()
					state.Reader = nil
					state.Mu.Unlock()

					return
				}

				return
			}

			if n == 0 {
				continue
			}

			written, writeErr := writeToRingBufferWithRetry(r.ctx, r.host, state.ReadBufferPtr, state.ReadBufferDataSize, buffer[:n], nil)
			if writeErr != nil {
				if writeErr == io.EOF {
					r.registry.CloseStream(r.ctx, r.host, r.streamID)
				}
				return
			}
			if written != n {
				return
			}
		}
	}()
}
