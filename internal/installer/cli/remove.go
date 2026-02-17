// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
)

// HandleRemove обрабатывает команду remove.
func (inst *Installer) HandleRemove(ctx context.Context, args []string, noCascade bool, dryRun bool) (err error) {

	if len(args) == 0 {
		return errors.New(i18n.Msg("Package not specified for removal"))
	}

	if dryRun {
		// Симуляция удаления - показываем что будет удалено
		for _, packageName := range args {
			var installations []models.Installation
			if installations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to get list of installations: %w"), err)
			}

			var foundInstallation *models.Installation
			for i := range installations {
				if installations[i].Package == packageName {
					foundInstallation = &installations[i]
					break
				}
			}

			if foundInstallation == nil {
				return fmt.Errorf(i18n.Msg("Package %s is not installed"), packageName)
			}

			fmt.Printf(i18n.Msg("Package will be removed: %s@%s")+"\n", foundInstallation.Package, foundInstallation.Version)
		}
		return
	}

	// Поддерживаем удаление нескольких пакетов
	for _, packageName := range args {
		if err = inst.removeSinglePackage(ctx, packageName, noCascade); err != nil {
			return fmt.Errorf(i18n.Msg("Error removing package %s: %w"), packageName, err)
		}
	}

	return
}

// removeSinglePackage удаляет один пакет.
func (inst *Installer) removeSinglePackage(ctx context.Context, packageName string, noCascade bool) (err error) {

	var installations []models.Installation
	if installations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get list of installations: %w"), err)
	}

	var foundInstallation *models.Installation
	for i := range installations {
		if installations[i].Package == packageName {
			foundInstallation = &installations[i]
			break
		}
	}

	if foundInstallation == nil {
		return fmt.Errorf(i18n.Msg("Package %s is not installed"), packageName)
	}

	// Удаляем пакет (файлы всегда удаляются согласно архитектуре)
	if err = inst.installationManager.Uninstall(ctx, foundInstallation.ID, false); err != nil {
		return
	}

	// Если каскадное удаление не отключено, удаляем зависимости, если они больше нигде не используются
	if !noCascade {
		if err = inst.removeUnusedDependencies(ctx, foundInstallation); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to remove unused dependencies: %w"), err)
		}
	}

	return
}

// removeUnusedDependencies удаляет зависимости удаленного пакета, если они больше нигде не используются.
func (inst *Installer) removeUnusedDependencies(ctx context.Context, removedInstallation *models.Installation) (err error) {

	if len(removedInstallation.Dependencies) == 0 {
		return
	}

	var allInstallations []models.Installation
	if allInstallations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get list of installations: %w"), err)
	}

	// Для каждой зависимости удаленного пакета проверяем, используется ли она другими пакетами
	for _, depStr := range removedInstallation.Dependencies {
		var parsedURI uri.URI
		if parsedURI, err = uri.New(depStr); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to parse dependency %s: %w"), depStr, err)
		}
		dep := models.Dependency{
			Source:  parsedURI.Source(),
			Package: parsedURI.Package(),
			Version: parsedURI.Version().Original,
		}
		if !inst.isDependencyUsed(ctx, dep, allInstallations) {
			// Зависимость не используется другими пакетами - удаляем её
			var depInstallation *models.Installation
			if depInstallation, err = inst.databaseManager.FindByPackage(ctx, dep.Source, dep.Package); err != nil {
				// Зависимость не установлена или уже удалена - пропускаем
				continue
			}

			// Рекурсивно удаляем зависимость
			if err = inst.installationManager.Uninstall(ctx, depInstallation.ID, false); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to remove unused dependency %s: %w"), dep.Package, err)
			}

			// Рекурсивно удаляем зависимости этой зависимости
			if err = inst.removeUnusedDependencies(ctx, depInstallation); err != nil {
				return
			}
		}
	}

	return
}

func (inst *Installer) isDependencyUsed(ctx context.Context, dep models.Dependency, allInstallations []models.Installation) (used bool) {

	for _, installation := range allInstallations {
		for _, installationDepStr := range installation.Dependencies {
			var parsedURI uri.URI
			var parseErr error
			if parsedURI, parseErr = uri.New(installationDepStr); parseErr != nil {
				continue
			}
			if parsedURI.Package() == dep.Package && parsedURI.Source() == dep.Source {
				return true
			}
		}
	}

	return
}
