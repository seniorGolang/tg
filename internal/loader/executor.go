// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package loader

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

type PluginExecutor interface {
	Execute(ctx context.Context, rootDir string, request plugin.Storage, path ...string) (response plugin.Storage, err error)
	Close(ctx context.Context) (err error)
}

type wasmExecutor struct {
	host *host.Host
}

func newWasmExecutor(host *host.Host) (executor *wasmExecutor) {

	return &wasmExecutor{
		host: host,
	}
}

func (e *wasmExecutor) Execute(ctx context.Context, rootDir string, request plugin.Storage, path ...string) (response plugin.Storage, err error) {

	return imports.Execute(ctx, e.host, rootDir, request, path...)
}

func (e *wasmExecutor) Close(ctx context.Context) (err error) {

	return wasm.Close(ctx, e.host)
}
