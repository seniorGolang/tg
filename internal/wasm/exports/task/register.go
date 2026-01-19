// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package task

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// RegisterTaskFunctions регистрирует все функции управления задачами в модуле env.
func RegisterTaskFunctions(builder wazero.HostModuleBuilder, h *host.Host) {

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, intervalMs, handlerID, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {
			return HostStartTask(ctx, h, intervalMs, handlerID, resultPtrPtr, resultSizePtr)
		}).
		Export("host_start_task")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, taskIDPtr, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {
			return HostStopTask(ctx, h, taskIDPtr, resultPtrPtr, resultSizePtr)
		}).
		Export("host_stop_task")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {
			return HostStopAll(ctx, h, resultPtrPtr, resultSizePtr)
		}).
		Export("host_stop_all_tasks")
}
