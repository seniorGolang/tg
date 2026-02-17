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

// ExecuteInitGenerator выполняет генерацию для плагина с InitPkgs.
func ExecuteInitGenerator(ctx context.Context, pluginLoader pluginLoader, pluginName string, rootDir string, moduleName string) (err error) {

	var installation *models.Installation
	if installation, err = pluginLoader.GetInfo(pluginName); err != nil {
		err = fmt.Errorf(i18n.Msg("plugin %s not found: %w"), pluginName, err)
		return
	}

	if len(installation.InitPkgs) == 0 {
		return
	}

	var ok bool
	var dbLoader *loader.DatabasePluginLoader
	if dbLoader, ok = pluginLoader.(*loader.DatabasePluginLoader); !ok {
		err = fmt.Errorf("%s", i18n.Msg("loader is not DatabasePluginLoader"))
		return
	}

	var wasmHost *host.Host
	if wasmHost, err = dbLoader.LoadHost(pluginName, rootDir, true); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to load plugin host: %w"), err)
		return
	}
	defer wasm.Close(ctx, wasmHost)

	err = imports.Generate(ctx, wasmHost, wasmRootDir, moduleName)
	return
}

// ExecuteInitCleanup выполняет очистку для плагина с InitPkgs.
func ExecuteInitCleanup(ctx context.Context, pluginLoader pluginLoader, pluginName string, rootDir string) (err error) {

	var installation *models.Installation
	if installation, err = pluginLoader.GetInfo(pluginName); err != nil {
		err = fmt.Errorf(i18n.Msg("plugin %s not found: %w"), pluginName, err)
		return
	}

	if len(installation.InitPkgs) == 0 {
		return
	}

	var ok bool
	var dbLoader *loader.DatabasePluginLoader
	if dbLoader, ok = pluginLoader.(*loader.DatabasePluginLoader); !ok {
		err = fmt.Errorf("%s", i18n.Msg("loader is not DatabasePluginLoader"))
		return
	}

	var wasmHost *host.Host
	if wasmHost, err = dbLoader.LoadHost(pluginName, rootDir, true); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to load plugin host: %w"), err)
		return
	}
	defer wasm.Close(ctx, wasmHost)

	err = imports.Generate(ctx, wasmHost, wasmRootDir, "")
	return
}
