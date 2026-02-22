// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"io"
	"time"

	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// StreamWriter реализует io.Writer для записи данных в поток через кольцевой буфер.
// StreamWriter: WASM пишет в кольцевой буфер, хост читает из буфера и пишет в io.Writer.
type Writer struct {
	streamID uint32
	registry registry
	host     memory.Host
	ctx      context.Context
}

func NewStreamWriter(ctx context.Context, streamID uint32, registry registry, h memory.Host) (writer *Writer) {
	return &Writer{
		streamID: streamID,
		registry: registry,
		host:     h,
		ctx:      ctx,
	}
}

// Write: хост читает из кольцевого буфера (куда пишет WASM) и записывает в io.Writer.
func (w *Writer) Write(buffer []byte) (n int, err error) {

	state, ok := w.registry.GetStreamState(w.streamID)
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

	if len(writeBuffer) < len(buffer) {
		writeBuffer = make([]byte, len(buffer))
	} else {
		writeBuffer = writeBuffer[:len(buffer)]
	}

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

	totalWritten := 0
	for totalWritten < readN {
		var writeErr error
		var writtenBytes int
		if writtenBytes, writeErr = writer.Write(writeBuffer[totalWritten:readN]); writeErr != nil {
			return 0, writeErr
		}

		if writtenBytes == 0 {
			break
		}

		totalWritten += writtenBytes
	}

	return totalWritten, nil
}

// StartWriter запускает горутину для непрерывного чтения данных из кольцевого буфера
// и записи их в Writer.
func (w *Writer) StartWriter() {

	go func() {
		buffer := make([]byte, 4096)
		for {
			select {
			case <-w.ctx.Done():
				return
			default:
			}

			state, ok := w.registry.GetStreamState(w.streamID)
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

			// В StartWriter нет канала уведомлений, поэтому передаем nil
			var n int
			var err error
			if n, err = readFromRingBufferWithRetry(w.ctx, w.host, state.WriteBufferPtr, state.WriteBufferDataSize, buffer, nil); err != nil {
				if err == io.EOF {
					w.registry.CloseStream(w.ctx, w.host, w.streamID)
				}
				return
			}

			totalWritten := 0
			for totalWritten < n {
				select {
				case <-w.ctx.Done():
					return
				default:
				}

				var writeErr error
				var writtenBytes int
				if writtenBytes, writeErr = writer.Write(buffer[totalWritten:n]); writeErr != nil {
					return
				}

				if writtenBytes == 0 {
					// Writer вернул 0 без ошибки — пауза и повтор в следующей итерации (уступка планировщику)
					select {
					case <-w.ctx.Done():
						return
					case <-time.After(time.Millisecond * 1):
						continue
					}
				}

				totalWritten += writtenBytes
			}
		}
	}()
}
