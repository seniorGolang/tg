// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"time"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

type executionPlan struct {
	Current     int        `json:"current"`
	Steps       []planStep `json:"steps"`
	CommandPath []string   `json:"commandPath"`
	CommandArgs []string   `json:"commandArgs"`
}

type planStep struct {
	Name         string        `json:"name"`
	Kind         string        `json:"kind"`
	Version      string        `json:"version,omitempty"`
	RequestKeys  []string      `json:"requestKeys,omitempty"`
	ResponseKeys []string      `json:"responseKeys,omitempty"`
	Duration     time.Duration `json:"duration,omitempty"`
}

func isSystemKey(key string) (result bool) {

	return key == executionPlanKey ||
		key == commandStatusKey ||
		key == commandStatusSuccess ||
		key == commandStatusError
}

func getStorageKeys(storage plugin.Storage) (keys []string) {

	if storage == nil {
		return
	}

	var ok bool
	var mapStorage *plugin.MapStorage
	if mapStorage, ok = storage.(*plugin.MapStorage); ok {
		keys = make([]string, 0, len(*mapStorage))
		for key := range *mapStorage {
			if !isSystemKey(key) {
				keys = append(keys, key)
			}
		}
	}

	return
}

func (plan Plan) toExecutionPlan(currentStepIndex int, currentStep *Step, pluginVersion string, requestKeys []string, responseKeys []string, duration time.Duration) (result executionPlan) {

	result.Steps = make([]planStep, 0, len(plan.Steps))
	result.Current = currentStepIndex

	for i, step := range plan.Steps {
		var stepInfo planStep
		stepInfo.Name = step.Name
		stepInfo.Kind = string(step.Kind)
		stepInfo.Version = step.PluginVersion

		if i < currentStepIndex {
			var keysInfo stepKeysInfo
			var exists bool
			keysInfo, exists = plan.StepKeys[step.Name]
			if exists {
				stepInfo.RequestKeys = keysInfo.RequestKeys
				stepInfo.ResponseKeys = keysInfo.ResponseKeys
				stepInfo.Duration = keysInfo.Duration
				if keysInfo.Version != "" {
					stepInfo.Version = keysInfo.Version
				}
			}
		} else if i == currentStepIndex {
			if currentStep != nil {
				stepInfo.RequestKeys = requestKeys
				stepInfo.ResponseKeys = responseKeys
				stepInfo.Duration = duration
				if pluginVersion != "" {
					stepInfo.Version = pluginVersion
				}
			} else {
				var keysInfo stepKeysInfo
				var exists bool
				keysInfo, exists = plan.StepKeys[step.Name]
				if exists {
					stepInfo.RequestKeys = keysInfo.RequestKeys
					stepInfo.ResponseKeys = keysInfo.ResponseKeys
					stepInfo.Duration = keysInfo.Duration
					if keysInfo.Version != "" {
						stepInfo.Version = keysInfo.Version
					}
				}
			}
		}

		result.Steps = append(result.Steps, stepInfo)
	}

	result.CommandPath = plan.CommandPath
	result.CommandArgs = plan.CommandArgs

	return
}
