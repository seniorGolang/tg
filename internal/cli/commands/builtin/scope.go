// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
)

func HandlePluginScopeCreate(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	options := make(map[string]any)
	if from, ok := ctx.Options[optionKeyFrom]; ok {
		options[optionKeyFrom] = from
	}
	if config, ok := ctx.Options[optionKeyConfig]; ok {
		options[optionKeyConfig] = config
	}

	err = installer.HandleScopeCreate(cmdCtx, ctx.Args, options)
	return
}

func HandlePluginScopeUse(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	err = installer.HandleScopeUse(cmdCtx, ctx.Args)
	return
}

func HandlePluginScopeList(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	err = installer.HandleScopeList(cmdCtx)
	return
}

func HandlePluginScopeDelete(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	force := hasOption(ctx, optionKeyForce)

	err = installer.HandleScopeDelete(cmdCtx, ctx.Args, force)
	return
}

func HandlePluginScopeShow(ctx types.CommandContext) (err error) {

	var cmdCtx context.Context
	if cmdCtx, err = prepareInstallerContext(ctx); err != nil {
		return
	}

	err = installer.HandleScopeShow(cmdCtx, ctx.Args)
	return
}
