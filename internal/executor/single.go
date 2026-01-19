// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/state"
)

func (e *Executor) ExecutePluginWithOptions(pluginName string, options map[string]any) (result ExecutionResult, err error) {

	startTime := time.Now()

	rootDir := e.stateManager.RootDir

	pluginOptions := make(map[string]any, len(options))
	for k, v := range options {
		pluginOptions[k] = v
	}

	var cmdPath []string
	var pathVal []string
	var ok bool
	if pathVal, ok = options[optionKeyCommandPath].([]string); ok {
		cmdPath = pathVal
	}

	request := plugin.NewStorage()

	for k, v := range pluginOptions {
		if err = request.Set(k, v); err != nil {
			err = fmt.Errorf(i18n.Msg("error setting option %s: %w"), k, err)
			return
		}
	}

	select {
	case <-e.ctx.Done():
		err = fmt.Errorf(i18n.Msg("execution cancelled: %w"), e.ctx.Err())
		return
	default:
	}

	var executor loader.PluginExecutor
	if executor, err = e.loader.LoadExecutor(pluginName); err != nil {
		result = ExecutionResult{
			PluginName: pluginName,
			Success:    false,
			Error:      err,
			Duration:   time.Since(startTime),
		}
		return
	}
	defer func() {
		if closeErr := executor.Close(e.ctx); closeErr != nil {
			if e.logger != nil {
				e.logger.Warn(i18n.Msg("failed to close plugin executor"), "plugin", pluginName, "error", closeErr)
			}
		}
	}()

	var response plugin.Storage
	if response, err = executor.Execute(e.ctx, rootDir, request, cmdPath...); err != nil {
		// Игнорируем response, так как он не используется в этом методе
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

		if err = e.stateManager.SetPluginState(pluginName, pluginOptions, stateResult); err != nil {
			if e.logger != nil {
				e.logger.Warn(i18n.Msg("failed to update plugin state in cache"), "plugin", pluginName, "error", err)
			}
		} else {
			if e.logger != nil {
				e.logger.Debug(i18n.Msg("plugin state updated in cache"), "plugin", pluginName)
			}
		}
	}

	return
}
