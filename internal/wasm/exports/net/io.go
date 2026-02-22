// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func connClose(ctx context.Context, h *host.Host, nm *netManager, connID uint64) (result uint64) {

	var err error
	var state *connState
	if state, err = nm.GetConnState(connID); err == nil && state != nil && state.streamID != 0 {
		if h.StreamRegistry != nil {
			h.StreamRegistry.CloseStream(ctx, h, state.streamID)
		}
	}

	var conn net.Conn
	if conn, err = nm.GetConn(connID); err != nil {
		slog.Error(i18n.Msg("ConnClose: failed to get conn"), "error", err, "connID", connID)
		return writeError(ctx, h, err)
	}

	nm.DelConn(connID)

	if err = conn.Close(); err != nil {
		slog.Error(i18n.Msg("ConnClose: failed to close conn"), "error", err, "connID", connID)
		return writeError(ctx, h, err)
	}

	return 0
}

// connGetBufferPtr для обратной совместимости возвращает ReadBufferPtr (хост → WASM).
// WASM должен использовать этот указатель для чтения данных из кольцевого буфера.
// Для записи данных WASM должен использовать WriteBufferPtr через отдельную функцию (если будет добавлена).
func connGetBufferPtr(ctx context.Context, h *host.Host, nm *netManager, connID uint64, bufferPtrPtr uint32) (result uint64) {

	var err error
	var state *connState
	if state, err = nm.GetConnState(connID); err != nil {
		return writeError(ctx, h, err)
	}

	if state.streamID == 0 {
		return writeError(ctx, h, errors.New(i18n.Msg("connection does not use streaming")))
	}

	var ok bool
	var bufferPtr uint32
	if h.StreamRegistry != nil {
		if bufferPtr, ok = h.StreamRegistry.GetBufferPtr(state.streamID); !ok {
			return writeError(ctx, h, errors.New(i18n.Msg("stream buffer not found")))
		}
	} else {
		return writeError(ctx, h, errors.New(i18n.Msg("stream registry not available")))
	}

	var mem api.Memory
	if mem = h.Module.Memory(); mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	if !mem.WriteUint32Le(bufferPtrPtr, bufferPtr) {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write buffer pointer")))
	}

	return 0
}

// connGetWriteBufferPtr: буфер для записи (WASM → хост).
func connGetWriteBufferPtr(ctx context.Context, h *host.Host, nm *netManager, connID uint64, bufferPtrPtr uint32) (result uint64) {

	var err error
	var state *connState
	if state, err = nm.GetConnState(connID); err != nil {
		return writeError(ctx, h, err)
	}

	if state.streamID == 0 {
		return writeError(ctx, h, errors.New(i18n.Msg("connection does not use streaming")))
	}

	var bufferPtr uint32
	if h.StreamRegistry != nil {
		var ok bool
		if bufferPtr, ok = h.StreamRegistry.GetWriteBufferPtr(state.streamID); !ok {
			return writeError(ctx, h, errors.New(i18n.Msg("stream write buffer not found")))
		}
	} else {
		return writeError(ctx, h, errors.New(i18n.Msg("stream registry not available")))
	}

	var mem api.Memory
	if mem = h.Module.Memory(); mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	if !mem.WriteUint32Le(bufferPtrPtr, bufferPtr) {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write buffer pointer")))
	}

	return 0
}
