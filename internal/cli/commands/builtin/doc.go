// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/cli"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/markdown"
)

func HandleDoc(ctx types.CommandContext) (err error) {

	var pluginNameArg string
	if len(ctx.Args) > 0 {
		pluginNameArg = ctx.Args[0]
	}

	var version string
	var pluginName string
	var selectedItem *docItem

	if pluginNameArg == "" {
		if selectedItem, pluginName, version, err = handleInteractiveMode(ctx); err != nil {
			return
		}
		if selectedItem == nil {
			return
		}
	} else {
		if selectedItem, pluginName, version, err = handleDirectMode(ctx, pluginNameArg); err != nil {
			return
		}
	}

	checkVersionMismatch(ctx, version, selectedItem, pluginName)

	if pluginNameArg == "" {
		logCommand(ctx, pluginName, selectedItem)
	}

	if selectedItem.doc == "" {
		ctx.Logger.Warn(i18n.Msg("Documentation for plugin not found"), "plugin", pluginName)
		return
	}

	var rendered string
	if rendered, err = markdown.RenderContent(selectedItem.doc, markdown.WithWidthPercent(100)); err != nil {
		return
	}

	fmt.Println(rendered)
	return
}

func handleInteractiveMode(ctx types.CommandContext) (selectedItem *docItem, pluginName string, version string, err error) {

	var items []*docItem
	if items, err = collectDocItems(ctx); err != nil {
		failedMsg := i18n.Msg("Failed to collect plugins and packages: ") + err.Error()
		return nil, "", "", errors.New(failedMsg)
	}

	if len(items) == 0 {
		ctx.Logger.Warn(i18n.Msg("No available plugins"))
		return
	}

	if selectedItem, err = selectDocItemInteractive(items, ctx); err != nil {
		return nil, "", "", errors.New(i18n.Msg("Selected plugin not found"))
	}
	if selectedItem == nil {
		return
	}

	pluginName = selectedItem.name
	version = selectedItem.version

	if selectedItem.doc == "" {
		var scope string
		if ctx.GlobalOpts.Scope != "" {
			scope = ctx.GlobalOpts.Scope
		}
		pluginDoc := getPluginDoc(ctx, scope, selectedItem.name, selectedItem.version)
		if pluginDoc != "" {
			selectedItem.doc = pluginDoc
		}
	}

	return
}

func handleDirectMode(ctx types.CommandContext, pluginNameArg string) (selectedItem *docItem, pluginName string, version string, err error) {

	pluginName, version = parsePluginArg(pluginNameArg)

	var scope string
	if ctx.GlobalOpts.Scope != "" {
		scope = ctx.GlobalOpts.Scope
	}
	var inst *cli.Installer
	if inst, err = cli.NewInstaller(scope); err != nil {
		notFoundMsg := i18n.Msg("Plugin not found") + errorSeparator + pluginName
		return nil, "", "", errors.New(notFoundMsg)
	}

	cmdCtx := context.Background()
	var installations []models.Installation
	if installations, err = inst.DatabaseManager().ListInstallations(cmdCtx); err != nil {
		notFoundMsg := i18n.Msg("Plugin not found") + errorSeparator + pluginName
		return nil, "", "", errors.New(notFoundMsg)
	}

	foundInstallation := findInstallation(installations, pluginName, version)

	if foundInstallation == nil {
		if selectedItem, err = handleNotInstalled(ctx, inst, pluginName, version, scope); err != nil {
			return
		}
	} else {
		if selectedItem, err = handleInstalled(ctx, inst, pluginName, version, foundInstallation, scope); err != nil {
			return
		}
	}

	return
}

func handleNotInstalled(ctx types.CommandContext, inst *cli.Installer, pluginName string, version string, scope string) (selectedItem *docItem, err error) {

	normalizedVersion := strings.TrimPrefix(version, versionPrefixV)
	var pkg *models.Package
	var manifest *models.Manifest
	if pkg, manifest, err = findPackageInCatalog(ctx, inst, pluginName, normalizedVersion); err != nil {
		notFoundMsg := i18n.Msg("Plugin not found") + errorSeparator + pluginName
		return nil, errors.New(notFoundMsg)
	}

	selectedItem = createDocItemFromPackage(pkg, manifest, nil)

	pluginDoc := getPluginDoc(ctx, scope, pluginName, manifest.Version)
	if pluginDoc != "" {
		selectedItem.doc = pluginDoc
	}

	return
}

func handleInstalled(ctx types.CommandContext, inst *cli.Installer, pluginName string, version string, foundInstallation *models.Installation, scope string) (selectedItem *docItem, err error) {

	var pkg *models.Package
	if pkg, _, err = findPackageWithSelection(ctx, inst, foundInstallation.Package, foundInstallation); err != nil {
		failedMsg := i18n.Msg("Failed to get package information: ") + err.Error()
		return nil, errors.New(failedMsg)
	}

	selectedItem = createDocItemFromPackage(pkg, nil, foundInstallation)

	pluginDoc := getPluginDoc(ctx, scope, pluginName, foundInstallation.Version)
	if pluginDoc != "" {
		selectedItem.doc = pluginDoc
	}

	return
}

func checkVersionMismatch(ctx types.CommandContext, version string, selectedItem *docItem, pluginName string) {

	if version == "" {
		return
	}

	normalizedRequested := strings.TrimPrefix(version, versionPrefixV)
	normalizedSelected := strings.TrimPrefix(selectedItem.version, versionPrefixV)
	if normalizedSelected != normalizedRequested {
		ctx.Logger.Warn(i18n.Msg("Installed plugin version differs from requested"),
			"plugin", pluginName,
			"installed", selectedItem.version,
			"requested", version)
		ctx.Logger.Info(i18n.Msg("Showing documentation for installed version"))
	}
}

func logCommand(ctx types.CommandContext, pluginName string, selectedItem *docItem) {

	pluginArg := pluginName
	if selectedItem.version != "" {
		pluginArg = pluginName + versionSeparator + selectedItem.version
	}
	cmdParts := []string{cmdNameTG, cmdPathPluginDoc, cmdSubPluginDoc, pluginArg}
	fullCmd := strings.Join(cmdParts, commandPathSeparator)

	slog.Info(i18n.Msg("run"), "cmd", fullCmd)
}
