// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"
	"errors"
	"math"
	"net"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func connSetDeadline(ctx context.Context, h *host.Host, nm *netManager, connID uint64, deadline uint64) (result uint64) {

	var err error
	var conn net.Conn
	if conn, err = nm.GetConn(connID); err != nil {
		return writeError(ctx, h, err)
	}

	// deadline передается как наносекунды (UnixNano)
	// Если deadline = 0, это означает сброс deadline (IsZero)
	var deadlineTime time.Time
	if deadline > 0 {
		if deadline > uint64(math.MaxInt64) {
			return writeError(ctx, h, errors.New(i18n.Msg("deadline too large")))
		}
		deadlineTime = time.Unix(0, int64(deadline))
	}

	if err = conn.SetDeadline(deadlineTime); err != nil {
		return writeError(ctx, h, err)
	}

	return 0
}

func connSetReadDeadline(ctx context.Context, h *host.Host, nm *netManager, connID uint64, deadline uint64) (result uint64) {

	var err error
	var conn net.Conn
	if conn, err = nm.GetConn(connID); err != nil {
		return writeError(ctx, h, err)
	}

	// deadline передается как наносекунды (UnixNano)
	// Если deadline = 0, это означает сброс deadline (IsZero)
	var deadlineTime time.Time
	if deadline > 0 {
		if deadline > uint64(math.MaxInt64) {
			return writeError(ctx, h, errors.New(i18n.Msg("deadline too large")))
		}
		deadlineTime = time.Unix(0, int64(deadline))
	}

	if err = conn.SetReadDeadline(deadlineTime); err != nil {
		return writeError(ctx, h, err)
	}

	return 0
}

func connSetWriteDeadline(ctx context.Context, h *host.Host, nm *netManager, connID uint64, deadline uint64) (result uint64) {

	var err error
	var conn net.Conn
	if conn, err = nm.GetConn(connID); err != nil {
		return writeError(ctx, h, err)
	}

	// deadline передается как наносекунды (UnixNano)
	// Если deadline = 0, это означает сброс deadline (IsZero)
	var deadlineTime time.Time
	if deadline > 0 {
		if deadline > uint64(math.MaxInt64) {
			return writeError(ctx, h, errors.New(i18n.Msg("deadline too large")))
		}
		deadlineTime = time.Unix(0, int64(deadline))
	}

	if err = conn.SetWriteDeadline(deadlineTime); err != nil {
		return writeError(ctx, h, err)
	}

	return 0
}
