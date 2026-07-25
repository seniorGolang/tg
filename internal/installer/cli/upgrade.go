// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

// HandleUpgrade заменяет пакеты последней доступной версией: после замены остаётся ровно одна версия.
func (inst *Installer) HandleUpgrade(ctx context.Context, args []string) (err error) {

	var installations []models.Installation
	if installations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get list of installations: %w"), err)
	}

	groups := groupInstallations(installations)
	keys := selectGroups(groups, args)
	if len(keys) == 0 {
		return errors.New(i18n.Msg("No packages to upgrade"))
	}

	for _, key := range keys {
		if err = inst.upgradePackage(ctx, key, groups[key]); err != nil {
			return
		}
	}

	return
}

// upgradePackage устанавливает последнюю доступную версию пакета и удаляет остальные версии.
func (inst *Installer) upgradePackage(ctx context.Context, key packageKey, group []models.Installation) (err error) {

	var target *managers.PackageWithSource
	if target, err = inst.latestAvailable(ctx, key); err != nil {
		return
	}

	var targetVersion models.Version
	if targetVersion, err = version.Parse(target.Manifest.Version); err != nil {
		return fmt.Errorf(i18n.Msg("Invalid manifest version format: %w"), err)
	}

	installCtx := context.WithValue(ctx, ContextKeySource, target.Source)
	if err = inst.installationManager.Install(installCtx, target.Package, targetVersion); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to install package %s: %w"), key.name, err)
	}

	var removed []string
	if removed, err = inst.keepOnlyVersion(ctx, group, targetVersion); err != nil {
		return
	}

	// Published skills адресуются по имени, поэтому снятие старых версий стирает общий путь: переустанавливаем skills выжившей версии.
	if err = inst.installationManager.InstallSkills(ctx, []string{key.name}); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to install skills: %w"), err)
	}

	if len(removed) == 0 {
		pterm.Info.Printf(i18n.Msg("Package %s is already at version %s")+"\n", key.name, targetVersion.Original)
		return
	}
	pterm.Success.Printf(i18n.Msg("Package %s upgraded to %s, removed versions: %s")+"\n", key.name, targetVersion.Original, strings.Join(removed, ", "))

	return
}

// latestAvailable ищет в каталоге самую новую версию пакета для источника установки.
func (inst *Installer) latestAvailable(ctx context.Context, key packageKey) (found *managers.PackageWithSource, err error) {

	var available []managers.PackageWithSource
	if available, err = inst.manifestManager.FindAllPackages(ctx, key.name); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to find package %s: %w"), key.name, err)
	}

	var latest models.Version
	for i := range available {
		if available[i].Manifest == nil || storage.BaseSourceURL(available[i].Source) != key.source {
			continue
		}
		var parsed models.Version
		var parseErr error
		if parsed, parseErr = version.Parse(available[i].Manifest.Version); parseErr != nil {
			continue
		}
		if found == nil || version.Compare(parsed, latest) > 0 {
			found = &available[i]
			latest = parsed
		}
	}

	if found == nil {
		return nil, fmt.Errorf(i18n.Msg("Package %s not found"), key.name)
	}

	return
}
