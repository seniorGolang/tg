// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package loader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/logger"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/cache"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
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

	if l == nil {
		return nil
	}
	return l.installations
}

func (l *DatabasePluginLoader) GetInfo(packageName string) (installation *models.Installation, err error) {

	var exists bool
	installation, exists = l.installations[packageName]
	if !exists {
		installation = nil
		err = fmt.Errorf(i18n.Msg("package %s not found"), packageName)
		return
	}

	if !l.isPlugin(installation) {
		installation = nil
		err = fmt.Errorf(i18n.Msg("package %s is not a plugin"), packageName)
		return
	}

	return
}

func (l *DatabasePluginLoader) LoadExecutor(name string, rootDir string) (wasmHost *host.Host, err error) {

	var exists bool
	var installation *models.Installation
	installation, exists = l.installations[name]
	if !exists {
		wasmHost = nil
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
		wasmHost = nil
		err = fmt.Errorf(i18n.Msg("plugin %s has no .tgp file"), name)
		return
	}

	var rawBytes []byte
	if rawBytes, err = os.ReadFile(wasmFilePath); err != nil {
		wasmHost = nil
		err = fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), err)
		return
	}

	var wasmBytes []byte
	if wasmBytes, err = plugin.DecodeTGPBytes(rawBytes); err != nil {
		wasmHost = nil
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
		AllowedListeners: installation.AllowedListeners,
		AllowedShellCMDs: installation.AllowedShellCMDs,
		AllowedStdOut:    installation.AllowedStdOut,
		AllowedStdErr:    installation.AllowedStdErr,
	}

	var tgPath string
	if scopeConfig, scopeErr := storage.LoadScopeConfig(l.scopeName); scopeErr == nil && scopeConfig != nil {
		tgPath = scopeConfig.ConfigDir
	}

	ctx := context.Background()
	opts := []wasm.Option{wasm.WithTGPath(tgPath)}

	compilationCache, cacheErr := cache.GetCompilationCache(ctx)
	if cacheErr != nil {
		slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), "error", cacheErr)
	}
	if compilationCache != nil {
		opts = append(opts, wasm.WithCompilationCache(compilationCache))
	}

	if wasmHost, err = wasm.New(ctx, wasmBytes, info, rootDir, loggerAdapter, opts...); err != nil {
		wasmHost = nil
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", err)
		return
	}

	return
}

func (l *DatabasePluginLoader) LoadHost(name string, rootDir string, useInitPkgs bool) (wasmHost *host.Host, err error) {

	var exists bool
	var installation *models.Installation
	installation, exists = l.installations[name]
	if !exists {
		wasmHost = nil
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
		wasmHost = nil
		err = fmt.Errorf(i18n.Msg("plugin %s has no .tgp file"), name)
		return
	}

	var rawBytes []byte
	if rawBytes, err = os.ReadFile(wasmFilePath); err != nil {
		wasmHost = nil
		err = fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), err)
		return
	}

	var wasmBytes []byte
	if wasmBytes, err = plugin.DecodeTGPBytes(rawBytes); err != nil {
		wasmHost = nil
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
		AllowedListeners: installation.AllowedListeners,
		AllowedShellCMDs: installation.AllowedShellCMDs,
		AllowedStdOut:    installation.AllowedStdOut,
		AllowedStdErr:    installation.AllowedStdErr,
	}

	var tgPath string
	if scopeConfig, scopeErr := storage.LoadScopeConfig(l.scopeName); scopeErr == nil && scopeConfig != nil {
		tgPath = scopeConfig.ConfigDir
	}

	ctx := context.Background()
	opts := []wasm.Option{wasm.WithTGPath(tgPath)}

	compilationCache, cacheErr := cache.GetCompilationCache(ctx)
	if cacheErr != nil {
		slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), "error", cacheErr)
	}
	if compilationCache != nil {
		opts = append(opts, wasm.WithCompilationCache(compilationCache))
	}

	if wasmHost, err = wasm.New(ctx, wasmBytes, info, rootDir, loggerAdapter, opts...); err != nil {
		wasmHost = nil
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", err)
		return
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

	return
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

func LoadInfoFromTGP(ctx context.Context, scopeName string, tgpPath string) (info plugin.Info, err error) {

	var rawBytes []byte
	if rawBytes, err = os.ReadFile(tgpPath); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), err)
		return
	}

	var wasmBytes []byte
	if wasmBytes, err = plugin.DecodeTGPBytes(rawBytes); err != nil {
		return
	}

	loggerAdapter := logger.NewSlogAdapter(slog.Default())

	var tgPath string
	if scopeConfig, scopeErr := storage.LoadScopeConfig(scopeName); scopeErr == nil && scopeConfig != nil {
		tgPath = scopeConfig.ConfigDir
	}

	compilationCache, cacheErr := cache.GetCompilationCache(ctx)
	if cacheErr != nil {
		slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), "error", cacheErr)
	}

	opts := []wasm.Option{wasm.WithTGPath(tgPath), wasm.MuteLogs()}
	if compilationCache != nil {
		opts = append(opts, wasm.WithCompilationCache(compilationCache))
	}

	var wasmHost *host.Host
	if wasmHost, err = wasm.New(ctx, wasmBytes, plugin.Info{}, ".", loggerAdapter, opts...); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", err)
		return
	}
	defer func() { _ = wasm.Close(ctx, wasmHost) }()

	if info, err = imports.Info(ctx, wasmHost); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to get plugin info: %w"), err)
		return
	}

	return
}
