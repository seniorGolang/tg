// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

func (m *manager) checkFileConflicts(ctx context.Context, pkg *models.Package, source string, versionStr string) (err error) {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var scopeConfig *storage.ScopeConfig
	if scopeConfig, err = storage.LoadScopeConfig(m.scopeName); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to load scope configuration: %w"), err)
	}

	var installations []models.Installation
	if installations, err = m.databaseManager.ListInstallations(ctx); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get list of installations: %w"), err)
	}

	destinationMap := make(map[string]string)

	for _, fileInst := range pkg.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		destination := m.resolveDestination(fileInst.Destination, scopeConfig)
		destinationMap[destination] = pkg.Name
	}

	force := false
	if forceVal := ctx.Value(contextkeys.Force); forceVal != nil {
		if f, ok := forceVal.(bool); ok {
			force = f
		}
	}

	for _, inst := range installations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// При --force игнорируем конфликты с установками того же пакета и версии
		// (источник не важен, так как при установке произойдет замена)
		if force && inst.Package == pkg.Name {
			var installedVersion models.Version
			var versionToCheck models.Version
			var parseErr1, parseErr2 error

			if installedVersion, parseErr1 = version.Parse(inst.Version); parseErr1 == nil {
				if versionToCheck, parseErr2 = version.Parse(versionStr); parseErr2 == nil {
					if version.Compare(installedVersion, versionToCheck) == 0 {
						continue
					}
				}
			}
		}

		for _, file := range inst.Files {
			if existingPackageName, exists := destinationMap[file.Path]; exists {
				return fmt.Errorf(i18n.Msg("File conflict: file %s is already installed by package %s, and package %s is also trying to install it"), file.Path, inst.Source+"/"+inst.Package, existingPackageName)
			}
		}
	}

	return
}
