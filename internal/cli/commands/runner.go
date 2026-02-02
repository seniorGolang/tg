// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/executor"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/state"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func getEffectiveOptions(cmd Command, planner *executor.Planner) (effectiveOptions []Option) {

	if planner != nil {
		if pluginCmd, ok := cmd.(*lazyPluginCommand); ok {
			var merged []models.OptionInfo
			var err error
			if merged, err = planner.GetMergedOptionsForCommand(pluginCmd.metadata.pluginName, pluginCmd.metadata.command.Path); err != nil {
				slog.Error(i18n.Msg("Failed to get merged options for command"), "error", err)
				effectiveOptions = cmd.GetOptions()
				return
			}
			effectiveOptions = convertOptionInfoToOptions(merged)
			return
		}
	}

	effectiveOptions = cmd.GetOptions()
	return
}

func createCommandRunner(cmd Command, rootDir string, planner *executor.Planner) (runner func(cobraCmd *cobra.Command, args []string)) {
	return func(cobraCmd *cobra.Command, args []string) {
		effectiveOptions := getEffectiveOptions(cmd, planner)
		rootCmd := cobraCmd.Root()
		if rootCmd.PersistentFlags().Changed("version") {
			if pluginCmd, ok := cmd.(*lazyPluginCommand); ok {
				var pluginVersion string
				var err error
				if pluginVersion, err = pluginCmd.GetPluginVersion(); err != nil {
					slog.Error(i18n.Msg("Failed to get plugin version"), "error", err)
					return
				}
				pluginName := pluginCmd.metadata.pluginName
				versionText := i18n.Msg("Version")
				pterm.Print(pterm.Green(versionText), " ", pterm.Cyan(pluginName), " ", pluginVersion, "\n")
				return
			}
			slog.Error(i18n.Msg("Version information is not available for builtin commands"))
			return
		}

		globalOpts := extractGlobalOptions(cobraCmd)
		logger := createLoggerWithLevel(globalOpts.LogLevel)
		updateGlobalSlogLevel(globalOpts.LogLevel)

		options := extractOptionsFromFlags(cobraCmd, effectiveOptions)
		commandPath := getCommandPath(cobraCmd)

		nonPositionalOptions := getNonPositionalOptions(effectiveOptions)
		hasOptions := len(nonPositionalOptions) > 0
		hasRequiredNonPositionalOptions := hasRequiredOptions(nonPositionalOptions)
		allOptionsProvided := validateRequiredOptions(options, nonPositionalOptions)

		if hasOptions && hasRequiredNonPositionalOptions && !allOptionsProvided {
			if globalOpts.FailOnMissing {
				requiredOpts := getRequiredOptions(nonPositionalOptions)
				slog.Error(i18n.Msg("Required options are missing"),
					"command", strings.Join(commandPath, " "),
					"required_options", requiredOpts)
				return
			}

			options = PromptCommandOptions(cmd, effectiveOptions, options, commandPath)
			if options == nil {
				return // Пользователь отменил
			}
		}

		positionalOptions := getPositionalOptions(effectiveOptions)
		hasRequiredPositional := hasRequiredOptions(positionalOptions)

		needsInteractiveArgs := false
		if hasRequiredPositional {
			requiredCount := 0
			for _, opt := range positionalOptions {
				if opt.Required {
					requiredCount++
				}
			}
			if len(args) < requiredCount {
				needsInteractiveArgs = true
			}
		}

		if cobraCmd.Args != nil {
			if err := cobraCmd.Args(cobraCmd, args); err != nil {
				needsInteractiveArgs = true
			}
		}

		if needsInteractiveArgs {
			if globalOpts.FailOnMissing {
				slog.Error(i18n.Msg("Required positional arguments are missing"),
					"command", strings.Join(commandPath, " "))
				return
			}

			args = PromptCommandArgs(cmd, cobraCmd, effectiveOptions, commandPath)
			if args == nil {
				return // Пользователь отменил
			}
		}

		if !globalOpts.HideCmd {
			showBuiltCommand(cmd, options, args, commandPath)
		}

		cmdCtx := cobraCmd.Context()
		if cmdCtx == nil {
			cmdCtx = context.Background()
		}

		ctx := CommandContext{
			Context:      cmdCtx,
			RootDir:      rootDir,
			Options:      options,
			Args:         args,
			CommandPath:  commandPath,
			Logger:       logger,
			StateManager: state.New(rootDir),
			GlobalOpts:   globalOpts,
		}

		if err := cmd.Execute(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			var pluginErr *imports.PluginError
			if errors.As(err, &pluginErr) {
				slog.Error(pluginErr.Message)
			} else {
				slog.Error(i18n.Msg("Command execution error"), "error", err)
			}
		}
	}
}
