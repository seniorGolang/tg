// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/installer/cli"
)

var installer *cli.Installer
var installerScope string

func initInstaller(scopeOverride ...string) (err error) {

	var scope string
	if len(scopeOverride) > 0 {
		scope = scopeOverride[0]
	}

	needsRecreate := installer == nil
	if installer != nil {
		if scope != installerScope {
			needsRecreate = true
		}
	}

	if needsRecreate {
		if scope != "" {
			if installer, err = cli.NewInstaller(scope); err != nil {
				return
			}
		} else if installer, err = cli.NewInstaller(); err != nil {
			return
		}
		installerScope = scope
	}

	return
}

func HandlePluginInstall(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	version := getStringOption(ctx, optionKeyVersion)
	force := getBoolOption(ctx, optionKeyForce)
	dryRun := getBoolOption(ctx, optionKeyDryRun)
	verbose := getBoolOption(ctx, optionKeyVerbose)

	return installer.HandleInstall(cmdCtx, ctx.Args, version, force, dryRun, verbose)
}

func HandlePluginRemove(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	noCascade := getBoolOption(ctx, optionKeyNoCascade)
	dryRun := getBoolOption(ctx, optionKeyDryRun)

	return installer.HandleRemove(cmdCtx, ctx.Args, noCascade, dryRun)
}

func HandlePluginList(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	return installer.HandleList(cmdCtx, ctx.Args)
}

func HandlePluginRepo(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	force := hasOption(ctx, optionKeyForce)

	return installer.HandleRepo(cmdCtx, ctx.Args, force)
}

func HandlePluginUpdate(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	force := hasOption(ctx, optionKeyForce)

	return installer.HandleUpdate(cmdCtx, ctx.Args, force)
}

func HandlePluginInfo(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	return installer.HandleInfo(cmdCtx, ctx.Args)
}

func HandlePluginSearch(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	return installer.HandleSearch(cmdCtx, ctx.Args)
}

func HandlePluginUpgradePackages(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	return installer.HandleUpgrade(cmdCtx, ctx.Args)
}
