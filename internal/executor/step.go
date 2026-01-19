// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func (e *Executor) executeStep(step *Step, plan *Plan, request plugin.Storage, hooks []stepHook) (response plugin.Storage, err error) {

	var installation *models.Installation
	if installation, err = e.loader.GetInfo(step.Name); err != nil {
		err = fmt.Errorf(i18n.Msg("plugin %s not found: %w"), step.Name, err)
		return
	}

	var executor loader.PluginExecutor
	if executor, err = e.loader.LoadExecutor(step.Name); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to load plugin executor: %w"), err)
		return
	}
	defer func() {
		if closeErr := executor.Close(e.ctx); closeErr != nil {
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
			err = fmt.Errorf(i18n.Msg("error in BeforeStep hook: %w"), err)
			return
		}
	}

	startTime := time.Now()

	if !step.Silent && e.logger != nil {
		e.logger.Debug(i18n.Msg("executing plugin"), "plugin", step.Name, "kind", step.Kind)
	}

	if response, err = executor.Execute(e.ctx, plan.RootDir, stepCtx.Request, plan.CommandPath...); err != nil {
		err = fmt.Errorf(i18n.Msg("error executing plugin %s: %w"), step.Name, err)
		return
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
