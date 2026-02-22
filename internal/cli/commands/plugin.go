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
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

type commandMetadata struct {
	command       models.CommandInfo
	pluginName    string
	wasmFilePath  string
	globalOptions []models.OptionInfo
}

type lazyPluginCommand struct {
	loader   pluginLoader
	metadata commandMetadata
}

type pluginLoader interface {
	GetInfo(packageName string) (installation *models.Installation, err error)
	LoadExecutor(name string, rootDir string) (wasmHost *host.Host, err error)
	GetList() (plugins []models.Installation, err error)
	GetInstallations() (installations map[string]*models.Installation)
}

func (c *lazyPluginCommand) GetPath() (path []string) {

	if c == nil {
		return nil
	}
	return c.metadata.command.Path
}

func (c *lazyPluginCommand) GetDescription() (description string) {

	if c == nil {
		return ""
	}
	return c.metadata.command.Description
}

// GetOptions: если у команды нет своих опций, подставляются глобальные опции плагина.
func (c *lazyPluginCommand) GetOptions() (options []Option) {

	cmdOptions := c.metadata.command.Options
	if len(cmdOptions) == 0 {
		cmdOptions = c.metadata.globalOptions
	}

	return convertOptionInfoToOptions(cmdOptions)
}

func (c *lazyPluginCommand) GetPluginVersion() (version string, err error) {

	var installation *models.Installation
	if installation, err = c.loader.GetInfo(c.metadata.pluginName); err != nil {
		return
	}

	return installation.Version, nil
}

func (c *lazyPluginCommand) Execute(ctx types.CommandContext) (err error) {

	exec := executor.NewExecutorWithContext(ctx.RootDir, ctx.Logger, ctx.Context, c.loader)
	planner := executor.NewPlanner(c.loader)

	mergedOptions := globalOptsToMap(ctx.GlobalOpts)
	for k, v := range ctx.Options {
		mergedOptions[k] = v
	}

	initialRequest := plugin.NewStorage()
	for k, v := range mergedOptions {
		if err = initialRequest.Set(k, v); err != nil {
			err = fmt.Errorf(i18n.Msg("error setting option %s: %w"), k, err)
			return
		}
	}

	commandArgs := buildCommandArgs(mergedOptions, ctx.Args)
	var plan executor.Plan
	if plan, err = planner.Plan(c.metadata.pluginName, initialRequest, ctx.RootDir, c.metadata.command.Path, commandArgs); err != nil {
		err = fmt.Errorf(i18n.Msg("error planning execution: %w"), err)
		return
	}

	return exec.ExecuteWithPlan(plan)
}

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

func registerPluginCommands(tree *CommandTree, loader pluginLoader) (err error) {

	var ok bool
	var dbLoader *pluginloader.DatabasePluginLoader
	if dbLoader, ok = loader.(*pluginloader.DatabasePluginLoader); !ok {
		err = fmt.Errorf("%s", i18n.Msg("loader is not DatabasePluginLoader"))
		return
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
