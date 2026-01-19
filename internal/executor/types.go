// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"context"
	"time"

	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/state"
)

type Plan struct {
	Steps          []Step
	InitialRequest plugin.Storage
	RootDir        string
	CommandPath    []string
	CommandArgs    []string
	StepKeys       map[string]stepKeysInfo
}

type Step struct {
	Name          string
	PluginVersion string
	Kind          Kind
	Dependencies  []string
	Silent        bool
}

type stepKeysInfo struct {
	RequestKeys  []string
	ResponseKeys []string
	Version      string
	Duration     time.Duration
}

type Executor struct {
	stateManager *state.Manager
	logger       plugin.Logger
	ctx          context.Context
	loader       pluginLoader
	hooks        []stepHook
}

type PluginTask struct {
	PluginName string
	Options    map[string]any
}

type ExecutionResult struct {
	PluginName string
	Success    bool
	Error      error
	Message    string
	Duration   time.Duration
}
