// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

type StepContext struct {
	Step          *Step
	Plan          *Plan
	Request       plugin.Storage
	PluginVersion string
	StartTime     time.Time
}

type StepHook interface {
	BeforeStep(ctx *StepContext) (err error)
	AfterStep(ctx *StepContext, response plugin.Storage, duration time.Duration) (err error)
}

type stepHook interface {
	beforeStep(ctx *StepContext) (err error)
	afterStep(ctx *StepContext, response plugin.Storage, duration time.Duration) (err error)
}

type planFormationHook struct {
	logger plugin.Logger
}

func newPlanFormationHook(logger plugin.Logger) (hook *planFormationHook) {

	return &planFormationHook{
		logger: logger,
	}
}

func findStepIndex(plan *Plan, step *Step) (index int) {

	index = -1
	for i := range plan.Steps {
		if plan.Steps[i].Name == step.Name && plan.Steps[i].Kind == step.Kind {
			index = i
			break
		}
	}

	return
}

func (h *planFormationHook) beforeStep(ctx *StepContext) (err error) {

	requestKeys := getStorageKeys(ctx.Request)
	currentStepIndex := findStepIndex(ctx.Plan, ctx.Step)

	execPlan := ctx.Plan.toExecutionPlan(currentStepIndex, ctx.Step, ctx.PluginVersion, requestKeys, nil, 0)

	if err = ctx.Request.Set(executionPlanKey, execPlan); err != nil {
		return
	}

	return
}

func (h *planFormationHook) afterStep(ctx *StepContext, response plugin.Storage, duration time.Duration) (err error) {

	requestKeys := getStorageKeys(ctx.Request)
	responseKeys := getStorageKeys(response)

	ctx.Plan.StepKeys[ctx.Step.Name] = stepKeysInfo{
		RequestKeys:  requestKeys,
		ResponseKeys: responseKeys,
		Version:      ctx.PluginVersion,
		Duration:     duration,
	}

	currentStepIndex := findStepIndex(ctx.Plan, ctx.Step)
	nextStepIndex := currentStepIndex + 1
	if nextStepIndex >= len(ctx.Plan.Steps) {
		nextStepIndex = currentStepIndex
	}

	execPlan := ctx.Plan.toExecutionPlan(nextStepIndex, nil, "", nil, nil, 0)

	if err = response.Set(executionPlanKey, execPlan); err != nil {
		h.logger.Warn(i18n.Msg("failed to update execution plan in response"), "plugin", ctx.Step.Name, "error", err)
		return
	}

	return
}
