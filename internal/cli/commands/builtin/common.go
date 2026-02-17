// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
)

func prepareInstallerContext(ctx types.CommandContext) (cmdCtx context.Context, err error) {

	var scope string
	if ctx.GlobalOpts.Scope != "" {
		scope = ctx.GlobalOpts.Scope
	}
	if err = initInstaller(scope); err != nil {
		return
	}

	cmdCtx = ctx.Context
	if cmdCtx == nil {
		cmdCtx = context.Background()
	}
	return
}

func getStringOption(ctx types.CommandContext, key string) (value string) {

	var ok bool
	var val any
	if val, ok = ctx.Options[key]; !ok {
		return
	}
	strVal, isString := val.(string)
	if !isString {
		return
	}
	value = strVal
	return
}

func getBoolOption(ctx types.CommandContext, key string) (value bool) {

	var ok bool
	var val any
	if val, ok = ctx.Options[key]; !ok {
		return
	}
	boolVal, isBool := val.(bool)
	if !isBool {
		return
	}
	value = boolVal
	return
}

func hasOption(ctx types.CommandContext, key string) (has bool) {

	_, has = ctx.Options[key]
	return
}
