// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/executor"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	pluginloader "github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

// commandMetadata содержит метаданные команды плагина.
type commandMetadata struct {
	pluginName    string
	command       models.CommandInfo
	globalOptions []models.OptionInfo
	wasmFilePath  string
}

// lazyPluginCommand представляет команду плагина с ленивой загрузкой.
type lazyPluginCommand struct {
	metadata commandMetadata
	loader   pluginLoader
}

// pluginLoader предоставляет интерфейс для загрузки плагинов.
// Использует интерфейс executor для совместимости.
type pluginLoader interface {
	GetInfo(packageName string) (installation *models.Installation, err error)
	LoadExecutor(name string) (executor pluginloader.PluginExecutor, err error)
	GetList() (plugins []models.Installation, err error)
	GetInstallations() (installations map[string]*models.Installation)
}

func (c *lazyPluginCommand) GetPath() (path []string) {

	return c.metadata.command.Path
}

func (c *lazyPluginCommand) GetDescription() (description string) {

	return c.metadata.command.Description
}

// GetOptions: если у команды нет своих опций, подставляются глобальные опции плагина.
func (c *lazyPluginCommand) GetOptions() (options []Option) {

	cmdOptions := c.metadata.command.Options
	if len(cmdOptions) == 0 {
		cmdOptions = c.metadata.globalOptions
	}

	options = convertOptionInfoToOptions(cmdOptions)
	return
}

func (c *lazyPluginCommand) GetPluginVersion() (version string, err error) {

	var installation *models.Installation
	if installation, err = c.loader.GetInfo(c.metadata.pluginName); err != nil {
		return
	}

	version = installation.Version
	return
}

// Execute выполняет команду плагина.
func (c *lazyPluginCommand) Execute(ctx types.CommandContext) (err error) {

	exec := executor.NewExecutorWithContext(ctx.RootDir, ctx.Logger, ctx.Context, c.loader)
	planner := executor.NewPlanner(c.loader)

	initialRequest := plugin.NewStorage()
	for k, v := range ctx.Options {
		if err = initialRequest.Set(k, v); err != nil {
			return fmt.Errorf(i18n.Msg("error setting option %s: %w"), k, err)
		}
	}

	commandArgs := buildCommandArgs(ctx.Options, ctx.Args, ctx.GlobalOpts)
	var plan executor.Plan
	if plan, err = planner.Plan(c.metadata.pluginName, initialRequest, ctx.RootDir, c.metadata.command.Path, commandArgs); err != nil {
		err = fmt.Errorf(i18n.Msg("error planning execution: %w"), err)
		return
	}

	if err = exec.ExecuteWithPlan(plan); err != nil {
		return
	}
	return
}

// convertOptionInfoToOptions конвертирует OptionInfo в Option для CLI.
func convertOptionInfoToOptions(optionInfos []models.OptionInfo) (options []Option) {

	options = make([]Option, 0, len(optionInfos))
	for _, optInfo := range optionInfos {
		options = append(options, Option{
			Name:         optInfo.Name,
			Short:        optInfo.Short,
			Type:         optInfo.Type,
			Description:  optInfo.Description,
			Required:     optInfo.Required,
			Default:      optInfo.Default,
			IsPositional: optInfo.IsPositional,
		})
	}

	return
}

// registerPluginCommands регистрирует команды из плагинов в дереве.
func registerPluginCommands(tree *CommandTree, loader pluginLoader) (err error) {

	var dbLoader *pluginloader.DatabasePluginLoader
	var ok bool
	if dbLoader, ok = loader.(*pluginloader.DatabasePluginLoader); !ok {
		return fmt.Errorf("%s", i18n.Msg("loader is not DatabasePluginLoader"))
	}

	installations := dbLoader.GetInstallations()

	for _, installation := range installations {
		if len(installation.Commands) == 0 {
			continue
		}

		var wasmFilePath string
		for _, file := range installation.Files {
			if strings.HasSuffix(file.Path, plugin.FileExtTGP) {
				wasmFilePath = file.Path
				break
			}
		}

		if wasmFilePath == "" {
			slog.Warn(i18n.Msg("plugin has no .tgp file, skipping commands"),
				"plugin", installation.Package)
			continue
		}

		for _, cmdInfo := range installation.Commands {
			metadata := commandMetadata{
				pluginName:    installation.Package,
				command:       cmdInfo,
				globalOptions: installation.Options,
				wasmFilePath:  wasmFilePath,
			}

			lazyCmd := &lazyPluginCommand{metadata: metadata, loader: loader}
			if err = tree.RegisterCommand(lazyCmd); err != nil {
				slog.Warn(i18n.Msg("failed to register command"),
					"plugin", installation.Package,
					"command", strings.Join(cmdInfo.Path, commandPathSeparator),
					"error", err)
				continue
			}
		}
	}

	return
}
