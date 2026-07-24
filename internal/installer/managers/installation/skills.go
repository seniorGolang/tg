// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/skills"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

func (m *manager) activatePackageSkills(ctx context.Context, pkg *models.Package, installation *models.Installation, installPrefix string) (err error) {

	opts := skills.FromContext(ctx)

	var roots []skills.Root
	if roots, err = skills.ResolveRoots(installPrefix, pkg.Name, pkg.Skills, installation.Skills); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to resolve skills: %w"), err)
	}
	if len(roots) == 0 {
		return
	}

	if existing, getErr := m.databaseManager.GetInstallation(ctx, installation.ID); getErr == nil && existing != nil && len(existing.Skills) > 0 {
		if err = skills.Deactivate(existing.Skills); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to deactivate previous skills: %w"), err)
		}
	}

	var home string
	if home, err = skills.Home(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to resolve home directory: %w"), err)
	}

	var skipped []string
	var states []models.SkillState
	if states, skipped, err = skills.Activate(home, roots, opts); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to activate skills: %w"), err)
	}
	for _, name := range skipped {
		slog.Warn(i18n.Msg("Skills target skipped: root directory not found"), slog.String("target", name))
	}

	installation.Skills = states
	return
}

func (m *manager) deactivateInstallationSkills(installation *models.Installation) (err error) {

	if installation == nil || len(installation.Skills) == 0 {
		return
	}
	if err = skills.Deactivate(installation.Skills); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to deactivate skills: %w"), err)
	}
	return
}

// InstallSkills публикует skills установленных пакетов.
func (m *manager) InstallSkills(ctx context.Context, packageNames []string) (err error) {

	opts := skills.FromContext(ctx)

	var scopeConfig *storage.ScopeConfig
	if scopeConfig, err = storage.LoadScopeConfig(m.scopeName); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to load scope configuration: %w"), err)
	}

	var installations []models.Installation
	if installations, err = m.databaseManager.ListInstallations(ctx); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to list installations: %w"), err)
	}

	filter := make(map[string]struct{}, len(packageNames))
	for _, name := range packageNames {
		filter[name] = struct{}{}
	}

	var home string
	if home, err = skills.Home(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to resolve home directory: %w"), err)
	}

	for i := range installations {
		installation := &installations[i]
		if len(filter) > 0 {
			if _, ok := filter[installation.Package]; !ok {
				continue
			}
		}

		var roots []skills.Root
		if roots, err = skills.ResolveRoots(scopeConfig.InstallPrefix, installation.Package, nil, installation.Skills); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to resolve skills: %w"), err)
		}
		if len(roots) == 0 {
			if roots, err = skills.Scan(scopeConfig.InstallPrefix, installation.Package); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to resolve skills: %w"), err)
			}
		}
		if len(roots) == 0 {
			continue
		}

		if err = skills.Deactivate(installation.Skills); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to deactivate skills: %w"), err)
		}

		var skipped []string
		var states []models.SkillState
		if states, skipped, err = skills.Activate(home, roots, opts); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to activate skills: %w"), err)
		}
		for _, name := range skipped {
			slog.Warn(i18n.Msg("Skills target skipped: root directory not found"), slog.String("target", name))
		}

		installation.Skills = states
		if err = m.databaseManager.RecordInstallation(ctx, installation); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to record installation in database: %w"), err)
		}
	}

	return
}
