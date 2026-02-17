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
	RootDir        string
	StepKeys       map[string]stepKeysInfo
	CommandPath    []string
	CommandArgs    []string
	InitialRequest plugin.Storage
}

type Step struct {
	Name          string
	Kind          Kind
	Silent        bool
	Dependencies  []string
	PluginVersion string
}

type stepKeysInfo struct {
	Version      string
	Duration     time.Duration
	RequestKeys  []string
	ResponseKeys []string
}

type Executor struct {
	ctx          context.Context
	hooks        []stepHook
	loader       pluginLoader
	logger       plugin.Logger
	stateManager *state.Manager
}

type PluginTask struct {
	Options    map[string]any
	PluginName string
}

type ExecutionResult struct {
	Error      error
	Success    bool
	Message    string
	Duration   time.Duration
	PluginName string
}
