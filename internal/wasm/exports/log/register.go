// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package log

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func RegisterLogFunctions(builder wazero.HostModuleBuilder, logger plugin.Logger, h *host.Host) {

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr uint32, msgLen uint32) {
			HostLog(ctx, logger, h, msgPtr, msgLen)
		}).
		Export("host_log")
}
