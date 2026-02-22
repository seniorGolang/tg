// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package interactive

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func RegisterInteractiveFunctions(builder wazero.HostModuleBuilder, h *host.Host) {

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, promptPtr uint32, promptLen uint32, optionsPtr uint32, optionsLen uint32, configPtr uint32, configLen uint32, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {
			return HostInteractiveSelect(ctx, h, promptPtr, promptLen, optionsPtr, optionsLen, configPtr, configLen, resultPtrPtr, resultSizePtr)
		}).
		Export("host_interactive_select")
}
