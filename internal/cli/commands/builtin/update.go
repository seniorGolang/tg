// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/executor"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/database"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/logger"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/state"

	"github.com/pterm/pterm"
)

func HandleUpdateWithPrompt(ctx types.CommandContext, promptOptionsFunc func([]types.Option, map[string]any, []string) map[string]any) (err error) {

	var states map[string]state.PluginState
	if states, err = ctx.StateManager.LoadAllStates(); err != nil {
		failedMsg := i18n.Msg("Failed to load states: ") + err.Error()
		return errors.New(failedMsg)
	}

	if len(states) == 0 {
		ctx.Logger.Info(i18n.Msg("No saved plugins to execute"))
		return
	}

	var tasks []executor.PluginTask
	if tasks, err = promptPluginSelection(states, ctx, promptOptionsFunc); err != nil {
		return
	}

	if len(tasks) == 0 {
		return
	}

	runPluginsParallel(tasks, ctx.RootDir, ctx.Logger)
	return
}

func promptPluginSelection(states map[string]state.PluginState, ctx types.CommandContext, promptOptionsFunc func([]types.Option, map[string]any, []string) map[string]any) (tasks []executor.PluginTask, err error) {

	ctx.Logger.Info(i18n.Msg("Saved plugins for execution:"))

	pluginOptions := make([]string, 0, len(states))
	pluginMap := make(map[string]string)

	for name := range states {
		optionText := name
		pluginOptions = append(pluginOptions, optionText)
		pluginMap[optionText] = name
	}

	selectedOptions, _ := pterm.DefaultInteractiveMultiselect.
		WithOptions(pluginOptions).
		WithDefaultOptions(pluginOptions).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(pluginOptions))).
		Show(i18n.Msg("Select plugins to execute"))

	selected := make([]string, 0, len(selectedOptions))
	for _, option := range selectedOptions {
		if name, ok := pluginMap[option]; ok {
			selected = append(selected, name)
		}
	}

	if len(selected) == 0 {
		pterm.Warning.Println(i18n.Msg("Nothing selected"))
		return
	}

	tasks = make([]executor.PluginTask, 0, len(selected))

	for _, name := range selected {
		pluginState, exists := states[name]
		if !exists {
			continue
		}

		pluginMsg := i18n.Msg("Plugin") + ": " + name
		ctx.Logger.Info(pluginMsg)
		ctx.Logger.Info(i18n.Msg("Parameters:"))
		for key, val := range pluginState.Options {
			paramMsg := paramIndent + key + paramSeparator + fmt.Sprintf("%v", val)
			ctx.Logger.Info(paramMsg)
		}

		change, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultValue(false).
			Show(i18n.Msg("Change parameters?"))

		options := pluginState.Options
		if change {
			if promptOptionsFunc != nil {
				options = promptOptionsFunc([]types.Option{}, options, []string{name})
			}
			if options == nil {
				continue
			}
		}

		tasks = append(tasks, executor.PluginTask{
			PluginName: name,
			Options:    options,
		})
	}

	return
}

func runPluginsParallel(tasks []executor.PluginTask, rootDir string, log plugin.Logger) {

	if len(tasks) == 0 {
		return
	}

	showParallelHeader()

	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()

	stateMgr := state.New(rootDir)

	var scopeName string
	var scopeErr error
	if scopeName, scopeErr = storage.GetCurrentScope(); scopeErr != nil {
		scopeName = storage.DefaultScopeName
	}
	dbManager := database.NewManager(scopeName)
	var pluginLoader *loader.DatabasePluginLoader
	var loaderErr error
	if pluginLoader, loaderErr = loader.New(scopeName, dbManager); loaderErr != nil {
		slog.Warn(fmt.Sprintf(i18n.Msg("Failed to create %s"), "plugin loader"), "error", loaderErr)
		return
	}

	pluginExecutors := make([]*pluginExecutor, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, task executor.PluginTask) {
			defer wg.Done()

			writer := multi.NewWriter()

			executingMsg := pluginNamePrefix + task.PluginName + pluginNameSuffix + i18n.Msg("Executing")
			bar, _ := pterm.DefaultProgressbar.
				WithTotal(progressBarTotal).
				WithTitle(executingMsg).
				WithWriter(writer).
				Start()

			logBuf := &logger.LogBuffer{
				PluginName: task.PluginName,
			}

			adapter := logger.NewBufferedLoggerAdapter(logBuf)

			exec := executor.NewExecutorWithContext(rootDir, adapter, context.Background(), pluginLoader)

			result, _ := exec.ExecutePluginWithOptions(task.PluginName, task.Options)

			for i := 0; i < progressBarTotal; i++ {
				bar.Increment()
			}
			completedMsg := pluginNamePrefix + task.PluginName + pluginNameSuffix + i18n.Msg("Completed")
			bar.UpdateTitle(completedMsg)
			_, _ = bar.Stop()

			pluginExecutors[idx] = &pluginExecutor{
				result:    result,
				bar:       bar,
				logBuffer: logBuf,
			}
		}(i, task)
	}

	wg.Wait()

	_, _ = multi.Stop()
	pterm.Println()

	for _, pe := range pluginExecutors {
		if pe != nil && pe.logBuffer != nil {
			logs := pe.logBuffer.GetLogs()
			if len(logs) > 0 {
				for _, logLine := range logs {
					pterm.Println(logLine)
				}
				pterm.Println()
			}
		}
	}

	results := make([]executor.ExecutionResult, len(tasks))
	for i, pe := range pluginExecutors {
		if pe != nil {
			results[i] = pe.result
		}
	}

	showResultsTable(results)

	err := stateMgr.SaveAllStates()
	if err != nil {
		slog.Warn(i18n.Msg("Failed to save states after parallel execution"), "error", err)
	}
}

type pluginExecutor struct {
	result    executor.ExecutionResult
	bar       *pterm.ProgressbarPrinter
	logBuffer *logger.LogBuffer
}

func showParallelHeader() {

	pterm.DefaultHeader.
		WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgWhite, pterm.Bold)).
		Println(i18n.Msg("Parallel plugin execution"))
	pterm.Println()
}

func showResultsTable(results []executor.ExecutionResult) {

	pterm.Println()

	tableData := pterm.TableData{
		{i18n.Msg("Plugin"), i18n.Msg("Status"), i18n.Msg("Message"), ""},
	}

	for _, result := range results {
		statusStr := i18n.Msg("✓ Success")
		if !result.Success {
			statusStr = i18n.Msg("✗ Error")
		}

		message := result.Message
		if len(message) > messageTruncateLength {
			message = message[:messageTruncateStart] + messageTruncateSuffix
		}

		durationStr := utils.FormatDuration(result.Duration)

		tableData = append(tableData, []string{
			result.PluginName,
			statusStr,
			message,
			durationStr,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithData(tableData).
		WithHeaderStyle(pterm.NewStyle(pterm.FgWhite, pterm.Bold, pterm.BgBlue)).
		WithRowSeparator(tableRowSeparator).
		WithHeaderRowSeparator(tableRowSeparator).
		Render() //nolint:errcheck
}
