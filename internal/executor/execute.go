// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func (e *Executor) ExecuteWithPlan(plan Plan) (err error) {

	currentRequest := plan.InitialRequest
	if currentRequest == nil {
		currentRequest = plugin.NewStorage()
	}

	if e.logger != nil {
		e.logger.Debug(i18n.Msg("plan execution started"), "steps", len(plan.Steps))
	}

	hooks := make([]stepHook, 0, len(e.hooks)+1)
	hooks = append(hooks, newPlanFormationHook(e.logger))
	hooks = append(hooks, e.hooks...)

	var commandStep *Step
	var commandStatus string
	commandStatus = commandStatusSuccess
	var postSteps []Step
	var preStageSteps []Step
	var commandResponse plugin.Storage
	var lastStageResponse plugin.Storage
	lastStageResponse = currentRequest

	for i := range plan.Steps {
		step := &plan.Steps[i]
		if step.Kind == KindCommand {
			commandStep = step
			break
		}
		if step.Kind != KindPost {
			preStageSteps = append(preStageSteps, *step)
		}
	}

	if commandStep == nil {
		err = errors.New(i18n.Msg("command step not found in plan"))
		return
	}

	for i := range plan.Steps {
		step := &plan.Steps[i]
		if step.Kind == KindPost {
			postSteps = append(postSteps, *step)
		}
	}

	for i := range preStageSteps {
		step := &preStageSteps[i]

		if e.logger != nil {
			e.logger.Debug(i18n.Msg("step execution"),
				"step", i+1,
				"total", len(preStageSteps)+1,
				"plugin", step.Name,
				"kind", step.Kind)
		}

		select {
		case <-e.ctx.Done():
			err = fmt.Errorf(i18n.Msg("execution cancelled: %w"), e.ctx.Err())
			return
		default:
		}

		var response plugin.Storage
		if response, err = e.executeStep(step, &plan, currentRequest, hooks); err != nil {
			return
		}

		currentRequest = response
		if step.Kind == KindStage {
			lastStageResponse = response
		}
	}

	if e.logger != nil {
		e.logger.Debug(i18n.Msg("command execution"),
			"plugin", commandStep.Name,
			"path", plan.CommandPath)
	}

	select {
	case <-e.ctx.Done():
		err = fmt.Errorf(i18n.Msg("execution cancelled: %w"), e.ctx.Err())
		return
	default:
	}

	var commandErr error
	if commandResponse, commandErr = e.executeStep(commandStep, &plan, currentRequest, hooks); commandErr != nil {
		commandStatus = commandStatusError
	}

	var postRequest plugin.Storage
	for i := range postSteps {
		step := &postSteps[i]

		if e.logger != nil {
			e.logger.Debug(i18n.Msg("post step execution"),
				"step", i+1,
				"total", len(postSteps),
				"plugin", step.Name)
		}

		select {
		case <-e.ctx.Done():
			err = fmt.Errorf(i18n.Msg("execution cancelled: %w"), e.ctx.Err())
			return
		default:
		}

		if i == 0 {
			if postRequest, err = buildPostRequest(lastStageResponse, commandStatus, commandResponse); err != nil {
				return
			}
		}

		var response plugin.Storage
		if response, err = e.executeStep(step, &plan, postRequest, hooks); err != nil {
			return
		}

		postRequest = response
	}

	if commandErr != nil {
		err = commandErr
		return
	}

	if e.logger != nil {
		e.logger.Debug(i18n.Msg("plan execution completed"))
	}
	return
}

// buildPostRequest: вход для post-хуков — последний stage + статус команды (success/error) + ответ команды.
func buildPostRequest(lastStageResponse plugin.Storage, commandStatus string, commandResponse plugin.Storage) (postRequest plugin.Storage, err error) {

	postRequest = plugin.NewStorage()

	if lastStageResponse != nil {
		var ok bool
		var mapStorage *plugin.MapStorage
		if mapStorage, ok = lastStageResponse.(*plugin.MapStorage); ok {
			for key, value := range *mapStorage {
				if err = postRequest.Set(key, value); err != nil {
					return
				}
			}
		}
	}

	if err = postRequest.Set(commandStatusKey, commandStatus); err != nil {
		return
	}

	if commandResponse != nil {
		var ok bool
		var mapStorage *plugin.MapStorage
		if mapStorage, ok = commandResponse.(*plugin.MapStorage); ok {
			for key, value := range *mapStorage {
				if err = postRequest.Set(key, value); err != nil {
					return
				}
			}
		}
	}

	return
}
