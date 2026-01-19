// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"log/slog"
	"runtime"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/executor"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	pluginloader "github.com/seniorGolang/tg/v3/internal/loader"
)

// ExecuteInitGenerators выполняет генерацию плагинов параллельно.
func ExecuteInitGenerators(ctx context.Context, loader pluginLoader, pluginNames []string, rootDir string, moduleName string) (err error) {

	if len(pluginNames) == 0 {
		return
	}

	maxConcurrency := runtime.NumCPU()
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, pluginName := range pluginNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			execLoader, ok := loader.(*pluginloader.DatabasePluginLoader)
			if !ok {
				slog.Warn(i18n.Msg("loader is not DatabasePluginLoader"), "plugin", name)
				return
			}

			if execErr := executor.ExecuteInitGenerator(ctx, execLoader, name, rootDir, moduleName); execErr != nil {
				slog.Warn(i18n.Msg("Generation error from plugin"),
					"plugin", name,
					"error", execErr)
			} else {
				slog.Info(i18n.Msg("Generation from plugin completed"),
					"plugin", name)
			}
		}(pluginName)
	}

	wg.Wait()
	return
}

// ExecuteInitCleanup выполняет очистку плагинов параллельно.
func ExecuteInitCleanup(ctx context.Context, loader pluginLoader, pluginNames []string, rootDir string) (err error) {

	if len(pluginNames) == 0 {
		return
	}

	maxConcurrency := runtime.NumCPU()
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, pluginName := range pluginNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			execLoader, ok := loader.(*pluginloader.DatabasePluginLoader)
			if !ok {
				slog.Warn(i18n.Msg("loader is not DatabasePluginLoader"), "plugin", name)
				return
			}

			if execErr := executor.ExecuteInitCleanup(ctx, execLoader, name, rootDir); execErr != nil {
				slog.Warn(i18n.Msg("Cleanup error from plugin"),
					"plugin", name,
					"error", execErr)
			} else {
				slog.Info(i18n.Msg("Cleanup from plugin completed"),
					"plugin", name)
			}
		}(pluginName)
	}

	wg.Wait()
	return
}
