// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/database"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

func getPluginDoc(ctx types.CommandContext, scope string, pluginName string, version string) (doc string) {

	if scope == "" {
		var scopeErr error
		scope, scopeErr = storage.GetEffectiveScope()
		if scopeErr != nil {
			scope = storage.DefaultScopeName
		}
	}

	dbManager := database.NewManager(scope)
	var err error
	var pluginLoader *loader.DatabasePluginLoader
	if pluginLoader, err = loader.New(scope, dbManager); err != nil {
		return ""
	}

	var installation *models.Installation
	if installation, err = pluginLoader.GetInfo(pluginName); err != nil {
		return ""
	}

	if version != "" {
		normalizedVersion := strings.TrimPrefix(version, versionPrefixV)
		instNormalizedVersion := strings.TrimPrefix(installation.Version, versionPrefixV)
		if instNormalizedVersion != normalizedVersion {
			return ""
		}
	}

	var host *host.Host
	if host, err = pluginLoader.LoadHost(pluginName, ctx.RootDir, false); err != nil {
		return ""
	}
	defer func() { _ = wasm.Close(context.Background(), host) }()

	cmdCtx := context.Background()
	var info plugin.Info
	if info, err = imports.Info(cmdCtx, host); err != nil {
		return ""
	}

	return info.Doc
}
