// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/state"
)

func NewExecutorWithContext(rootDir string, logger plugin.Logger, ctx context.Context, loader pluginLoader) (executor *Executor) {

	if ctx == nil {
		ctx = context.Background()
	}

	return &Executor{
		stateManager: state.New(rootDir),
		logger:       logger,
		ctx:          ctx,
		loader:       loader,
		hooks:        make([]stepHook, 0),
	}
}
