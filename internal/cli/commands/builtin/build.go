// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/build"
	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

func HandlePluginBuild(ctx types.CommandContext) (err error) {

	rootDir := ctx.RootDir
	out := getStringOption(ctx, optionKeyOut)
	if out == "" {
		out = "./dist"
	}
	outDir := out
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(rootDir, outDir)
	}

	overrideManifest := getStringOption(ctx, optionKeyOverrideManifest)
	if overrideManifest == "" {
		overrideManifest = "./manifest.overrides.yml"
	}
	if !filepath.IsAbs(overrideManifest) {
		overrideManifest = filepath.Join(rootDir, overrideManifest)
	}

	scopeName := ctx.GlobalOpts.Scope
	if scopeName == "" {
		scopeName, _ = storage.GetEffectiveScope()
	}
	if scopeName == "" {
		scopeName = storage.DefaultScopeName
	}

	params := build.Params{
		RootDir:           rootDir,
		OutDir:            outDir,
		Clean:             getBoolOption(ctx, optionKeyClean),
		OverrideManifest:  overrideManifest,
		Version:           getStringOption(ctx, optionKeyVersion),
		SkipVersionUpdate: getBoolOption(ctx, optionKeySkipVersionUpdate),
		ScopeName:         scopeName,
		OutWriter:         os.Stdout,
	}

	cmdCtx := ctx.Context
	if cmdCtx == nil {
		cmdCtx = context.Background()
	}

	return build.Run(cmdCtx, params)
}
