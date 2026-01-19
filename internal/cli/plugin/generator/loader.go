// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"github.com/seniorGolang/tg/v3/internal/installer/managers/database"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	pluginloader "github.com/seniorGolang/tg/v3/internal/loader"
)

type pluginLoader interface {
	GetInfo(packageName string) (installation *models.Installation, err error)
	GetList() (plugins []models.Installation, err error)
}

func createLoader(scopeName string) (loader pluginLoader, err error) {

	if scopeName == "" {
		var scopeErr error
		if scopeName, scopeErr = storage.GetCurrentScope(); scopeErr != nil {
			scopeName = storage.DefaultScopeName
		}
	}

	dbManager := database.NewManager(scopeName)
	var newLoader *pluginloader.DatabasePluginLoader
	if newLoader, err = pluginloader.New(scopeName, dbManager); err != nil {
		return
	}

	loader = newLoader
	return
}

func GetInitGenerators(loader pluginLoader) (pluginNames []string, err error) {

	var allInstallations []models.Installation
	if allInstallations, err = loader.GetList(); err != nil {
		return
	}

	pluginNames = make([]string, 0)
	for i := range allInstallations {
		inst := &allInstallations[i]
		if len(inst.InitPkgs) > 0 {
			pluginNames = append(pluginNames, inst.Package)
		}
	}

	return
}
