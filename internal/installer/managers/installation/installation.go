// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"fmt"
	"os"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

type manager struct {
	scopeName          string
	manifestManager    managers.ManifestManager
	dependencyResolver managers.DependencyResolver
	downloadManager    managers.DownloadManager
	validationEngine   managers.ValidationEngine
	databaseManager    managers.DatabaseManager
}

func NewManager(scopeName string, manifestManager managers.ManifestManager, dependencyResolver managers.DependencyResolver, downloadManager managers.DownloadManager, validationEngine managers.ValidationEngine, databaseManager managers.DatabaseManager) (mgr managers.InstallationManager) {
	return &manager{
		scopeName:          scopeName,
		manifestManager:    manifestManager,
		dependencyResolver: dependencyResolver,
		downloadManager:    downloadManager,
		validationEngine:   validationEngine,
		databaseManager:    databaseManager,
	}
}

// Uninstall удаляет пакет.
func (m *manager) Uninstall(ctx context.Context, packageID string, keepFiles bool) (err error) {

	var installation *models.Installation
	if installation, err = m.databaseManager.GetInstallation(ctx, packageID); err != nil {
		return fmt.Errorf(i18n.Msg("Installation not found: %w"), err)
	}

	var pkg *models.Package
	if installation.Source != "" {
		packageSpec := installation.Source + "/" + installation.Package
		var manifest *models.Manifest
		if pkg, manifest, err = m.manifestManager.FindPackage(ctx, packageSpec); err != nil {
			if pkg, manifest, err = m.manifestManager.FindPackage(ctx, installation.Package); err != nil {
				pkg = nil
			}
		}
		_ = manifest
	} else {
		var manifest *models.Manifest
		if pkg, manifest, err = m.manifestManager.FindPackage(ctx, installation.Package); err != nil {
			pkg = nil
		}
		_ = manifest
	}

	if pkg != nil && pkg.Scripts != nil && pkg.Scripts.PreUninstall != nil {
		var scopeConfig *storage.ScopeConfig
		if scopeConfig, err = storage.LoadScopeConfig(m.scopeName); err == nil {
			workDir := scopeConfig.InstallPrefix
			if err = m.executeScript(ctx, pkg.Scripts.PreUninstall, workDir, nil); err != nil {
				return fmt.Errorf(i18n.Msg("Error executing pre_uninstall script: %w"), err)
			}
		}
	}

	if err = m.deactivateInstallationSkills(installation); err != nil {
		return
	}

	// Согласно архитектуре, файлы всегда удаляются при удалении пакета
	// Параметр keepFiles оставлен для обратной совместимости, но всегда должен быть false
	for _, file := range installation.Files {
		if file.Path == "" {
			continue
		}
		var statErr error
		if _, statErr = os.Stat(file.Path); statErr == nil {
			var removeErr error
			if removeErr = os.Remove(file.Path); removeErr != nil {
				return fmt.Errorf(i18n.Msg("Failed to remove file %s: %w"), file.Path, removeErr)
			}
		} else if !os.IsNotExist(statErr) {
			// Игнорируем ошибку, если файл не существует
			// statErr содержит другую ошибку, но мы её игнорируем согласно логике
			_ = statErr
		}
	}

	if err = m.databaseManager.RemoveInstallation(ctx, packageID); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to remove installation from database: %w"), err)
	}

	if pkg != nil && pkg.Scripts != nil && pkg.Scripts.PostUninstall != nil {
		var scopeConfig *storage.ScopeConfig
		if scopeConfig, err = storage.LoadScopeConfig(m.scopeName); err == nil {
			workDir := scopeConfig.InstallPrefix
			if err = m.executeScript(ctx, pkg.Scripts.PostUninstall, workDir, nil); err != nil {
				return fmt.Errorf(i18n.Msg("Error executing post_uninstall script: %w"), err)
			}
		}
	}

	return
}
