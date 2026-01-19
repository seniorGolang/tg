// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package command

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// RegisterCommandFunctions регистрирует все функции модуля command.
func RegisterCommandFunctions(builder wazero.HostModuleBuilder, h *host.Host) {

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, commandPtr uint32, commandLen uint32, argsPtr uint32, argsLen uint32, workDirPtr uint32, workDirLen uint32, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {
			return HostExecuteCommand(ctx, h, commandPtr, commandLen, argsPtr, argsLen, workDirPtr, workDirLen, resultPtrPtr, resultSizePtr)
		}).
		Export("host_execute_command")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, streamID uint32, bufferPtrPtr uint32) (resultCode uint32) {
			return HostGetStreamReadBufferPtr(ctx, h, streamID, bufferPtrPtr)
		}).
		Export("host_get_stream_read_buffer_ptr")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, streamID uint32, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {
			return HostGetCommandResponse(ctx, h, streamID, resultPtrPtr, resultSizePtr)
		}).
		Export("host_get_command_response")
}
