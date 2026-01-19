// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/seniorGolang/tg/v3/internal/cli/commands/builtin"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/database"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/loader"

	"github.com/spf13/cobra"
)

var (
	globalTree    *CommandTree
	globalRootCmd *cobra.Command
)

// RegisterAllCommands регистрирует все команды (встроенные и из плагинов)
func RegisterAllCommands(rootCmd *cobra.Command, rootDir string) {

	globalRootCmd = rootCmd

	builtin.SetCompletionGenerator(&cobraCompletionAdapter{cmd: rootCmd})

	rootCmd.PersistentFlags().String(GlobalFlagLogLevel, logLevelInfo,
		i18n.Msg("Logging level (debug, info, warn, error)"))
	rootCmd.PersistentFlags().Bool(GlobalFlagHideCmd, false,
		i18n.Msg("Hide command output after interactive mode"))
	rootCmd.PersistentFlags().Bool(GlobalFlagFailOnMissing, false,
		i18n.Msg("Output error instead of interactive mode when required options are missing"))
	rootCmd.PersistentFlags().String(GlobalFlagScope, "",
		i18n.Msg("Scope to use for command execution (overrides current scope)"))

	tree := NewCommandTree()
	globalTree = tree

	if err := registerBuiltinCommands(tree); err != nil {
		slog.Error(i18n.Msg("Error registering builtin commands"), "error", err)
	}

	var scopeName string
	var scopeErr error
	if scopeName, scopeErr = storage.GetCurrentScope(); scopeErr != nil {
		scopeName = storage.DefaultScopeName
	}
	dbManager := database.NewManager(scopeName)
	var pluginLoader *loader.DatabasePluginLoader
	var loaderErr error
	if pluginLoader, loaderErr = loader.New(scopeName, dbManager); loaderErr != nil {
		slog.Warn(fmt.Sprintf(i18n.Msg("Failed to create %s"), "plugin loader, skipping plugin commands"), "error", loaderErr)
	} else {
		if err := registerPluginCommands(tree, pluginLoader); err != nil {
			slog.Error(i18n.Msg("Error registering plugin commands"), "error", err)
		}
	}

	buildCobraCommands(rootCmd, tree, rootDir)
}

// cobraCompletionAdapter адаптирует cobra.Command к интерфейсу CompletionGenerator
type cobraCompletionAdapter struct {
	cmd *cobra.Command
}

func (a *cobraCompletionAdapter) GenBashCompletionV2(writer any, includeDesc bool) (err error) {

	var ioWriter io.Writer
	var ok bool
	if ioWriter, ok = writer.(io.Writer); !ok {
		err = errors.New(i18n.Msg("Invalid writer type, expected io.Writer"))
		return
	}
	if err = a.cmd.GenBashCompletionV2(ioWriter, includeDesc); err != nil {
		return
	}
	return
}

func (a *cobraCompletionAdapter) GenZshCompletion(writer any) (err error) {

	var ioWriter io.Writer
	var ok bool
	if ioWriter, ok = writer.(io.Writer); !ok {
		err = errors.New(i18n.Msg("Invalid writer type, expected io.Writer"))
		return
	}
	if err = a.cmd.GenZshCompletion(ioWriter); err != nil {
		return
	}
	return
}

func (a *cobraCompletionAdapter) GenFishCompletion(writer any, includeDesc bool) (err error) {

	var ioWriter io.Writer
	var ok bool
	if ioWriter, ok = writer.(io.Writer); !ok {
		err = errors.New(i18n.Msg("Invalid writer type, expected io.Writer"))
		return
	}
	if err = a.cmd.GenFishCompletion(ioWriter, includeDesc); err != nil {
		return
	}
	return
}

func (a *cobraCompletionAdapter) GenPowerShellCompletion(writer any) (err error) {

	var ioWriter io.Writer
	var ok bool
	if ioWriter, ok = writer.(io.Writer); !ok {
		err = errors.New(i18n.Msg("Invalid writer type, expected io.Writer"))
		return
	}
	if err = a.cmd.GenPowerShellCompletion(ioWriter); err != nil {
		return
	}
	return
}

func (a *cobraCompletionAdapter) GetName() (name string) {

	return a.cmd.Name()
}

func GetCommandTree() (tree *CommandTree) {

	return globalTree
}

func GetRootCmd() (rootCmd *cobra.Command) {

	return globalRootCmd
}
