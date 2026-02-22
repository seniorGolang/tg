// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

func (e *Executor) executeStep(step *Step, plan *Plan, request plugin.Storage, hooks []stepHook) (response plugin.Storage, err error) {

	var installation *models.Installation
	if installation, err = e.loader.GetInfo(step.Name); err != nil {
		return nil, fmt.Errorf(i18n.Msg("plugin %s not found: %w"), step.Name, err)
	}

	var wasmHost *host.Host
	if wasmHost, err = e.loader.LoadExecutor(step.Name, e.stateManager.RootDir); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to load plugin executor: %w"), err)
	}
	defer func() {
		if closeErr := wasm.Close(e.ctx, wasmHost); closeErr != nil {
			if e.logger != nil {
				e.logger.Warn(i18n.Msg("failed to close plugin executor"), "plugin", step.Name, "error", closeErr)
			}
		}
	}()

	stepCtx := &StepContext{
		Step:          step,
		Plan:          plan,
		Request:       request,
		PluginVersion: installation.Version,
		StartTime:     time.Now(),
	}

	for _, hook := range hooks {
		if err = hook.beforeStep(stepCtx); err != nil {
			return nil, fmt.Errorf(i18n.Msg("error in BeforeStep hook: %w"), err)
		}
	}

	startTime := time.Now()

	if !step.Silent && e.logger != nil {
		e.logger.Debug(i18n.Msg("executing plugin"), "plugin", step.Name, "kind", step.Kind)
	}

	if err = stepCtx.Request.Set(optionKeyRunDir, plan.RootDir); err != nil {
		return nil, fmt.Errorf(i18n.Msg("error setting runDir: %w"), err)
	}

	if err = stepCtx.Request.Set(optionKeyCommandPath, plan.CommandPath); err != nil {
		return nil, fmt.Errorf(i18n.Msg("error setting command path: %w"), err)
	}

	if response, err = imports.Execute(e.ctx, wasmHost, stepCtx.Request); err != nil {
		return nil, fmt.Errorf(i18n.Msg("error executing plugin %s: %w"), step.Name, err)
	}

	duration := time.Since(startTime)

	for _, hook := range hooks {
		if hookErr := hook.afterStep(stepCtx, response, duration); hookErr != nil {
			if e.logger != nil {
				e.logger.Warn(i18n.Msg("error in AfterStep hook"), "plugin", step.Name, "error", hookErr)
			}
		}
	}

	if !step.Silent && e.logger != nil {
		e.logger.Debug(i18n.Msg("step completed successfully"),
			"plugin", step.Name,
			"duration", duration)
	}

	return
}
