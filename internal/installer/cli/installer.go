// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/database"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/dependency"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/download"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/installation"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/manifest"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/scope"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/validation"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

const (
	packagePathSeparator = "/"
	changeTypeNew        = "new"
	changeTypeUpdated    = "updated"
	changeTypeUnchanged  = "unchanged"
	manifestYAMLExt      = ".yaml"
	manifestYMLExt       = ".yml"
)

// Installer представляет установщик пакетов с всеми компонентами.
type Installer struct {
	scopeManager        managers.ScopeManager
	currentScope        string
	databaseManager     managers.DatabaseManager
	downloadManager     managers.DownloadManager
	manifestManager     managers.ManifestManager
	validationEngine    managers.ValidationEngine
	dependencyResolver  managers.DependencyResolver
	installationManager managers.InstallationManager
}

func NewInstaller(scopeOverride ...string) (inst *Installer, err error) {

	scopeMgr := scope.NewManager()
	ctx := context.Background()

	var currentScope string
	if len(scopeOverride) > 0 && scopeOverride[0] != "" {
		currentScope = scopeOverride[0]
	} else {
		if currentScope, err = scopeMgr.GetCurrentScope(ctx); err != nil {
			currentScope = storage.DefaultScopeName
		}
	}

	manifestMgr := manifest.NewManager(currentScope)
	databaseMgr := database.NewManager(currentScope)
	dependencyResolver := dependency.NewResolver(manifestMgr, databaseMgr)
	downloadMgr := download.NewManager()
	validationEngine := validation.NewEngine()
	installationMgr := installation.NewManager(
		currentScope,
		manifestMgr,
		dependencyResolver,
		downloadMgr,
		validationEngine,
		databaseMgr,
	)

	return &Installer{
		scopeManager:        scopeMgr,
		manifestManager:     manifestMgr,
		dependencyResolver:  dependencyResolver,
		downloadManager:     downloadMgr,
		validationEngine:    validationEngine,
		installationManager: installationMgr,
		databaseManager:     databaseMgr,
		currentScope:        currentScope,
	}, nil
}

func (inst *Installer) DatabaseManager() (mgr managers.DatabaseManager) {

	if inst == nil {
		return nil
	}
	return inst.databaseManager
}

func (inst *Installer) ManifestManager() (mgr managers.ManifestManager) {

	if inst == nil {
		return nil
	}
	return inst.manifestManager
}
