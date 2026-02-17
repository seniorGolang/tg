// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"log/slog"

	"github.com/seniorGolang/tg/v3/internal/logger"
	"github.com/seniorGolang/tg/v3/internal/plugin"

	"github.com/spf13/cobra"
)

// extractGlobalOptions извлекает глобальные опции из корневой команды
func extractGlobalOptions(cmd *cobra.Command) (globalOpts GlobalOptions) {

	rootCmd := cmd.Root()

	logLevel, _ := rootCmd.PersistentFlags().GetString(GlobalFlagLogLevel)
	if logLevel == "" {
		logLevel = logLevelInfo
	}

	hideCmd, _ := rootCmd.PersistentFlags().GetBool(GlobalFlagHideCmd)
	failOnMissing, _ := rootCmd.PersistentFlags().GetBool(GlobalFlagFailOnMissing)
	scope, _ := rootCmd.PersistentFlags().GetString(GlobalFlagScope)

	return GlobalOptions{
		LogLevel:      logLevel,
		HideCmd:       hideCmd,
		FailOnMissing: failOnMissing,
		Scope:         scope,
	}
}

func globalOptsToMap(opts GlobalOptions) (m map[string]any) {

	m = make(map[string]any, 4)
	if opts.LogLevel != "" {
		m[GlobalFlagLogLevel] = opts.LogLevel
	}
	if opts.HideCmd {
		m[GlobalFlagHideCmd] = true
	}
	if opts.FailOnMissing {
		m[GlobalFlagFailOnMissing] = true
	}
	if opts.Scope != "" {
		m[GlobalFlagScope] = opts.Scope
	}

	return m
}

func parseSlogLevel(level string) (slogLevel slog.Level) {

	switch level {
	case logLevelDebug:
		return slog.LevelDebug
	case logLevelWarn:
		return slog.LevelWarn
	case logLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func createLoggerWithLevel(level string) (log plugin.Logger) {

	if level == "" {
		level = logLevelInfo
	}
	return logger.NewSlogAdapter(slog.New(logger.NewPTermHandler(parseSlogLevel(level))))
}

func updateGlobalSlogLevel(level string) {
	slog.SetDefault(slog.New(logger.NewPTermHandler(parseSlogLevel(level))))
}
