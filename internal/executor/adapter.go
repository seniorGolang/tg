// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"time"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func (e *Executor) AddHook(hook StepHook) {

	e.hooks = append(e.hooks, &stepHookAdapter{hook: hook})
}

type stepHookAdapter struct {
	hook StepHook
}

func (a *stepHookAdapter) beforeStep(ctx *StepContext) (err error) {

	if a.hook == nil {
		return nil
	}
	return a.hook.BeforeStep(ctx)
}

func (a *stepHookAdapter) afterStep(ctx *StepContext, response plugin.Storage, elapsed time.Duration) (err error) {

	if a.hook == nil {
		return nil
	}
	return a.hook.AfterStep(ctx, response, elapsed)
}
