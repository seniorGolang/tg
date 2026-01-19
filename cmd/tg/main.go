// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/seniorGolang/tg/v3/internal/cli/commands"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/logger"
	"github.com/seniorGolang/tg/v3/internal/wasm/cache"

	"github.com/spf13/cobra"
)

func main() {

	slog.SetDefault(slog.New(logger.NewPTermHandler(slog.LevelInfo)))

	var wd string
	var err error
	if wd, err = os.Getwd(); err != nil {
		slog.Error(i18n.Msg("Failed to get current working directory"), "error", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Debug(i18n.Msg("Shutdown signal received, closing resources"))
		cancel()
	}()

	defer func() {
		if err = cache.CloseCompilationCache(context.Background()); err != nil {
			slog.Warn(i18n.Msg("Error closing compilation cache"), "error", err)
		}
	}()

	rootCmd := &cobra.Command{
		Use:   "tg",
		Short: i18n.Msg("TG (Tool Gateway) - extensible platform for executing tasks through plugins"),
		Long:  i18n.Msg("TG (Tool Gateway) - extensible platform for executing tasks through plugins. Allows running plugins for various project tasks"),
		Run:   runRoot,
	}

	rootCmd.SetContext(ctx)

	// Добавляем флаг --version как PersistentFlag, чтобы он был доступен для всех команд
	rootCmd.PersistentFlags().Bool("version", false, i18n.Msg("Show version information"))

	commands.RegisterAllCommands(rootCmd, wd)

	if err = rootCmd.ExecuteContext(ctx); err != nil {
		slog.Error(i18n.Msg("Error"), "error", err)
	}
}
