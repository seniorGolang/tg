// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/state"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

func (e *Executor) ExecutePluginWithOptions(pluginName string, options map[string]any) (result ExecutionResult, err error) {

	startTime := time.Now()

	rootDir := e.stateManager.RootDir

	pluginOptions := make(map[string]any, len(options))
	for k, v := range options {
		pluginOptions[k] = v
	}

	request := plugin.NewStorage()

	for k, v := range pluginOptions {
		if err = request.Set(k, v); err != nil {
			err = fmt.Errorf(i18n.Msg("error setting option %s: %w"), k, err)
			return
		}
	}

	if err = request.Set(optionKeyRunDir, rootDir); err != nil {
		err = fmt.Errorf(i18n.Msg("error setting runDir: %w"), err)
		return
	}

	select {
	case <-e.ctx.Done():
		err = fmt.Errorf(i18n.Msg("execution cancelled: %w"), e.ctx.Err())
		return
	default:
	}

	var wasmHost *host.Host
	if wasmHost, err = e.loader.LoadExecutor(pluginName, rootDir); err != nil {
		result = ExecutionResult{
			PluginName: pluginName,
			Success:    false,
			Error:      err,
			Duration:   time.Since(startTime),
		}
		return
	}
	defer func() {

		if closeErr := wasm.Close(e.ctx, wasmHost); closeErr != nil {
			if e.logger != nil {
				e.logger.Warn(i18n.Msg("failed to close plugin executor"), "plugin", pluginName, "error", closeErr)
			}
		}
	}()

	var response plugin.Storage
	if response, err = imports.Execute(e.ctx, wasmHost, request); err != nil {
		_ = response
	}

	duration := time.Since(startTime)

	result = ExecutionResult{
		PluginName: pluginName,
		Success:    err == nil,
		Error:      err,
		Duration:   duration,
	}

	if err != nil {
		result.Message = err.Error()
	} else {
		result.Message = i18n.Msg("execution completed successfully")
	}

	if result.Success {
		stateResult := state.PluginExecutionResult{
			Success: result.Success,
			Message: result.Message,
		}
		if result.Error != nil {
			stateResult.Error = result.Error.Error()
		}

		e.stateManager.SetPluginState(pluginName, pluginOptions, stateResult)
		if e.logger != nil {
			e.logger.Debug(i18n.Msg("plugin state updated in cache"), "plugin", pluginName)
		}
	}

	return
}
