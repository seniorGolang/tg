// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package loader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/tetratelabs/wazero"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/logger"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/cache"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

type DatabasePluginLoader struct {
	scopeName     string
	dbManager     managers.DatabaseManager
	installations map[string]*models.Installation
}

func New(scopeName string, dbManager managers.DatabaseManager) (loader *DatabasePluginLoader, err error) {

	var installations []models.Installation
	if installations, err = dbManager.ListInstallations(context.Background()); err != nil {
		return nil, err
	}

	installationCache := make(map[string]*models.Installation)
	for i := range installations {
		inst := &installations[i]
		installationCache[inst.Package] = inst
	}

	return &DatabasePluginLoader{
		scopeName:     scopeName,
		dbManager:     dbManager,
		installations: installationCache,
	}, nil
}

func (l *DatabasePluginLoader) GetInstallations() (installations map[string]*models.Installation) {

	return l.installations
}

func (l *DatabasePluginLoader) GetInfo(packageName string) (installation *models.Installation, err error) {

	var exists bool
	installation, exists = l.installations[packageName]
	if !exists {
		err = fmt.Errorf(i18n.Msg("package %s not found"), packageName)
		return
	}

	if !l.isPlugin(installation) {
		err = fmt.Errorf(i18n.Msg("package %s is not a plugin"), packageName)
		return
	}

	return
}

func (l *DatabasePluginLoader) LoadExecutor(name string) (executor PluginExecutor, err error) {

	var exists bool
	var installation *models.Installation
	installation, exists = l.installations[name]
	if !exists {
		err = fmt.Errorf(i18n.Msg("package %s not found"), name)
		return
	}

	var wasmFilePath string
	for _, file := range installation.Files {
		if strings.HasSuffix(file.Path, plugin.FileExtTGP) {
			wasmFilePath = file.Path
			break
		}
	}

	if wasmFilePath == "" {
		err = fmt.Errorf(i18n.Msg("plugin %s has no .tgp file"), name)
		return
	}

	var wasmBytes []byte
	var readErr error
	if wasmBytes, readErr = os.ReadFile(wasmFilePath); readErr != nil {
		err = fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), readErr)
		return
	}

	loggerAdapter := logger.NewSlogAdapter(slog.Default())

	info := plugin.Info{
		Name:             installation.Package,
		Version:          installation.Version,
		Commands:         convertCommandInfosToPluginCommands(installation.Commands),
		Options:          convertOptionInfosToPluginOptions(installation.Options),
		Kind:             installation.Kind,
		Silent:           installation.Silent,
		Always:           installation.Always,
		AllowedPaths:     installation.AllowedPaths,
		AllowedEnvVars:   installation.AllowedEnvVars,
		AllowedHosts:     installation.AllowedHosts,
		AllowedShellCMDs: installation.AllowedShellCMDs,
	}

	var tgPath string
	var scopeConfig *storage.ScopeConfig
	var scopeErr error
	scopeConfig, scopeErr = storage.LoadScopeConfig(l.scopeName)
	if scopeErr == nil && scopeConfig != nil {
		tgPath = scopeConfig.ConfigDir
	}

	ctx := context.Background()
	var compilationCache wazero.CompilationCache
	var cacheErr error
	compilationCache, cacheErr = cache.GetCompilationCache(ctx)
	if cacheErr != nil {
		slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), "error", cacheErr)
	}

	var wasmHost *host.Host
	var hostErr error
	if compilationCache != nil {
		if wasmHost, hostErr = wasm.New(ctx, wasmBytes, info, ".", loggerAdapter, wasm.WithCompilationCache(compilationCache), wasm.WithTGPath(tgPath)); hostErr != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", hostErr)
			return
		}
	} else {
		if wasmHost, hostErr = wasm.New(ctx, wasmBytes, info, ".", loggerAdapter, wasm.WithTGPath(tgPath)); hostErr != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", hostErr)
			return
		}
	}

	executor = newWasmExecutor(wasmHost)
	return
}

func (l *DatabasePluginLoader) LoadHost(name string, useInitPkgs bool) (wasmHost *host.Host, err error) {

	var exists bool
	var installation *models.Installation
	installation, exists = l.installations[name]
	if !exists {
		err = fmt.Errorf(i18n.Msg("package %s not found"), name)
		return
	}

	var wasmFilePath string
	for _, file := range installation.Files {
		if strings.HasSuffix(file.Path, plugin.FileExtTGP) {
			wasmFilePath = file.Path
			break
		}
	}

	if wasmFilePath == "" {
		err = fmt.Errorf(i18n.Msg("plugin %s has no .tgp file"), name)
		return
	}

	var wasmBytes []byte
	var readErr error
	if wasmBytes, readErr = os.ReadFile(wasmFilePath); readErr != nil {
		err = fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), readErr)
		return
	}

	loggerAdapter := logger.NewSlogAdapter(slog.Default())

	// При useInitPkgs ограничиваем доступ к ФС только пакетами из InitPkgs (для инициализации без полного доступа).
	allowedPaths := installation.AllowedPaths
	if useInitPkgs && len(installation.InitPkgs) > 0 {
		allowedPaths = make(map[string]string, len(installation.InitPkgs))
		for _, pkg := range installation.InitPkgs {
			allowedPaths["@root/"+pkg] = "w"
		}
	}

	info := plugin.Info{
		Name:             installation.Package,
		Version:          installation.Version,
		Commands:         convertCommandInfosToPluginCommands(installation.Commands),
		Options:          convertOptionInfosToPluginOptions(installation.Options),
		Kind:             installation.Kind,
		Silent:           installation.Silent,
		Always:           installation.Always,
		AllowedPaths:     allowedPaths,
		AllowedEnvVars:   installation.AllowedEnvVars,
		AllowedHosts:     installation.AllowedHosts,
		AllowedShellCMDs: installation.AllowedShellCMDs,
	}

	var tgPath string
	var scopeConfig *storage.ScopeConfig
	var scopeErr error
	scopeConfig, scopeErr = storage.LoadScopeConfig(l.scopeName)
	if scopeErr == nil && scopeConfig != nil {
		tgPath = scopeConfig.ConfigDir
	}

	ctx := context.Background()
	var compilationCache wazero.CompilationCache
	var cacheErr error
	compilationCache, cacheErr = cache.GetCompilationCache(ctx)
	if cacheErr != nil {
		slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), "error", cacheErr)
	}

	var hostErr error
	if compilationCache != nil {
		if wasmHost, hostErr = wasm.New(ctx, wasmBytes, info, ".", loggerAdapter, wasm.WithCompilationCache(compilationCache), wasm.WithTGPath(tgPath)); hostErr != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", hostErr)
			return
		}
	} else {
		if wasmHost, hostErr = wasm.New(ctx, wasmBytes, info, ".", loggerAdapter, wasm.WithTGPath(tgPath)); hostErr != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", hostErr)
			return
		}
	}

	return
}

func (l *DatabasePluginLoader) GetList() (plugins []models.Installation, err error) {

	plugins = make([]models.Installation, 0)
	for _, inst := range l.installations {
		if l.isPlugin(inst) {
			plugins = append(plugins, *inst)
		}
	}

	return
}

func (l *DatabasePluginLoader) isPlugin(installation *models.Installation) (isPlugin bool) {

	// Пакет является плагином, если у него есть команды
	if len(installation.Commands) > 0 {
		return true
	}

	// Пакет является плагином, если у него установлен флаг always
	if installation.Always {
		return true
	}

	// Пакет является плагином, если у него есть файл .tgp (WASM модуль)
	for _, file := range installation.Files {
		if strings.HasSuffix(file.Path, plugin.FileExtTGP) {
			return true
		}
	}

	return false
}

func convertCommandInfosToPluginCommands(commandInfos []models.CommandInfo) (commands []plugin.Command) {

	commands = make([]plugin.Command, 0, len(commandInfos))
	for _, cmdInfo := range commandInfos {
		commands = append(commands, plugin.Command{
			Path:        cmdInfo.Path,
			Description: cmdInfo.Description,
			Options:     convertOptionInfosToPluginOptions(cmdInfo.Options),
		})
	}

	return
}

func convertOptionInfosToPluginOptions(optionInfos []models.OptionInfo) (options []plugin.Option) {

	options = make([]plugin.Option, 0, len(optionInfos))
	for _, optInfo := range optionInfos {
		options = append(options, plugin.Option{
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
