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

func createLoggerWithLevel(level string) (log plugin.Logger) {

	// Преобразуем строковый уровень в slog.Level
	var slogLevel slog.Level
	switch level {
	case logLevelDebug:
		slogLevel = slog.LevelDebug
	case logLevelInfo:
		slogLevel = slog.LevelInfo
	case logLevelWarn:
		slogLevel = slog.LevelWarn
	case logLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Создаём новый handler с нужным уровнем
	handler := logger.NewPTermHandler(slogLevel)

	// Создаём новый логгер с этим handler
	slogLogger := slog.New(handler)

	// Создаём адаптер
	return logger.NewSlogAdapter(slogLogger)
}

// updateGlobalSlogLevel обновляет глобальный slog с указанным уровнем логирования
func updateGlobalSlogLevel(level string) {

	// Преобразуем строковый уровень в slog.Level
	var slogLevel slog.Level
	switch level {
	case logLevelDebug:
		slogLevel = slog.LevelDebug
	case logLevelInfo:
		slogLevel = slog.LevelInfo
	case logLevelWarn:
		slogLevel = slog.LevelWarn
	case logLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Создаём новый handler с нужным уровнем
	handler := logger.NewPTermHandler(slogLevel)

	// Обновляем глобальный логгер
	slog.SetDefault(slog.New(handler))
}
