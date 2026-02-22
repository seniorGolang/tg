// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"context"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

const wasmRootDir = "/"

func ExecuteInitGenerator(ctx context.Context, pluginLoader pluginLoader, pluginName string, rootDir string, moduleName string) (err error) {

	var installation *models.Installation
	if installation, err = pluginLoader.GetInfo(pluginName); err != nil {
		return fmt.Errorf(i18n.Msg("plugin %s not found: %w"), pluginName, err)
	}

	if len(installation.InitPkgs) == 0 {
		return
	}

	var ok bool
	var dbLoader *loader.DatabasePluginLoader
	if dbLoader, ok = pluginLoader.(*loader.DatabasePluginLoader); !ok {
		return fmt.Errorf("%s", i18n.Msg("loader is not DatabasePluginLoader"))
	}

	var wasmHost *host.Host
	if wasmHost, err = dbLoader.LoadHost(pluginName, rootDir, true); err != nil {
		return fmt.Errorf(i18n.Msg("failed to load plugin host: %w"), err)
	}
	defer wasm.Close(ctx, wasmHost)

	return imports.Generate(ctx, wasmHost, wasmRootDir, moduleName)
}

func ExecuteInitCleanup(ctx context.Context, pluginLoader pluginLoader, pluginName string, rootDir string) (err error) {

	var installation *models.Installation
	if installation, err = pluginLoader.GetInfo(pluginName); err != nil {
		return fmt.Errorf(i18n.Msg("plugin %s not found: %w"), pluginName, err)
	}

	if len(installation.InitPkgs) == 0 {
		return
	}

	var ok bool
	var dbLoader *loader.DatabasePluginLoader
	if dbLoader, ok = pluginLoader.(*loader.DatabasePluginLoader); !ok {
		return fmt.Errorf("%s", i18n.Msg("loader is not DatabasePluginLoader"))
	}

	var wasmHost *host.Host
	if wasmHost, err = dbLoader.LoadHost(pluginName, rootDir, true); err != nil {
		return fmt.Errorf(i18n.Msg("failed to load plugin host: %w"), err)
	}
	defer wasm.Close(ctx, wasmHost)

	return imports.Generate(ctx, wasmHost, wasmRootDir, "")
}
