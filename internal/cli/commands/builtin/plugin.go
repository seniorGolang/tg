// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/database"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

func getPluginDoc(ctx types.CommandContext, scope string, pluginName string, version string) (doc string) {

	if scope == "" {
		var scopeErr error
		scope, scopeErr = storage.GetCurrentScope()
		if scopeErr != nil {
			scope = storage.DefaultScopeName
		}
	}

	dbManager := database.NewManager(scope)
	pluginLoader, err := loader.New(scope, dbManager)
	if err != nil {
		return ""
	}

	installation, err := pluginLoader.GetInfo(pluginName)
	if err != nil {
		return ""
	}

	if version != "" {
		normalizedVersion := strings.TrimPrefix(version, versionPrefixV)
		instNormalizedVersion := strings.TrimPrefix(installation.Version, versionPrefixV)
		if instNormalizedVersion != normalizedVersion {
			return ""
		}
	}

	host, err := pluginLoader.LoadHost(pluginName, false)
	if err != nil {
		return ""
	}
	defer wasm.Close(context.Background(), host)

	cmdCtx := context.Background()
	info, err := imports.Info(cmdCtx, host)
	if err != nil {
		return ""
	}

	return info.Doc
}
