// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

const (
	comparisonEqual   = 0
	comparisonGreater = 1
	comparisonLess    = -1
)

// versionCheckResult представляет результат проверки версии.
type versionCheckResult struct {
	shouldInstall bool
	skipReason    string
}

func (m *manager) checkPackageVersion(ctx context.Context, pkgToCheck *models.Package, versionToCheck models.Version, versionConstraint string, allInstallations []models.Installation) (result versionCheckResult, err error) {

	slog.Debug(i18n.Msg("checkPackageVersion: checking package"), slog.String("package", pkgToCheck.Name), slog.String("version_to_install", versionToCheck.Original), slog.String("version_constraint", versionConstraint), slog.Int("total_installations", len(allInstallations)))

	force := false
	if forceVal := ctx.Value(contextkeys.Force); forceVal != nil {
		if f, ok := forceVal.(bool); ok {
			force = f
		}
	}

	// Если указан --force, принудительно устанавливаем пакет, даже если версия совпадает
	if force {
		slog.Debug(i18n.Msg("checkPackageVersion: force flag is set, proceeding with installation"))
		result = versionCheckResult{shouldInstall: true}
		return
	}

	var foundInstallations []models.Installation
	for i := range allInstallations {
		if allInstallations[i].Package == pkgToCheck.Name {
			foundInstallations = append(foundInstallations, allInstallations[i])
		}
	}

	if len(foundInstallations) == 0 {
		slog.Debug(i18n.Msg("checkPackageVersion: no installed package found, proceeding with installation"))
		result = versionCheckResult{shouldInstall: true}
		return
	}

	var latestInstalled *models.Installation
	var latestVersion models.Version
	var sameVersionInstalled *models.Installation

	for i := range foundInstallations {
		installedVersion, parseErr := version.Parse(foundInstallations[i].Version)
		if parseErr != nil {
			slog.Debug(i18n.Msg("checkPackageVersion: failed to parse installed version"), slog.String("version", foundInstallations[i].Version), slog.Any("error", parseErr))
			continue
		}

		comparison := version.Compare(installedVersion, versionToCheck)
		if comparison == comparisonEqual {
			sameVersionInstalled = &foundInstallations[i]
			slog.Debug(i18n.Msg("checkPackageVersion: found installed package with same version"), slog.String("package", sameVersionInstalled.Package), slog.String("version", sameVersionInstalled.Version))
		}

		if latestVersion.Original == "" || version.Compare(installedVersion, latestVersion) > 0 {
			latestVersion = installedVersion
			latestInstalled = &foundInstallations[i]
		}
	}

	if sameVersionInstalled != nil {
		slog.Debug(i18n.Msg("checkPackageVersion: exact version match, skipping installation"))
		result = versionCheckResult{
			shouldInstall: false,
			skipReason:    fmt.Sprintf(i18n.Msg("Package %s version %s is already installed. Skipping installation.")+"\n", pkgToCheck.Name, versionToCheck.Original),
		}
		return
	}

	if versionConstraint != "" && latestInstalled != nil && latestVersion.Original != "" {
		installedVersionStr := latestInstalled.Version
		installedVersionParsed, parseErr := version.Parse(installedVersionStr)
		if parseErr == nil {
			if version.Match(versionConstraint, installedVersionParsed) {
				slog.Debug(i18n.Msg("checkPackageVersion: installed version satisfies constraint, skipping installation"), slog.String("installed_version", installedVersionStr), slog.String("constraint", versionConstraint))
				result = versionCheckResult{
					shouldInstall: false,
					skipReason:    fmt.Sprintf(i18n.Msg("Package %s version %s satisfies requirement %s. Skipping installation.")+"\n", pkgToCheck.Name, installedVersionStr, versionConstraint),
				}
				return
			}
		}
	}

	if latestInstalled != nil && latestVersion.Original != "" {
		comparison := version.Compare(latestVersion, versionToCheck)
		slog.Debug(i18n.Msg("checkPackageVersion: version comparison"), slog.String("installed_version", latestInstalled.Version), slog.String("version_to_install", versionToCheck.Original), slog.Int("comparison", comparison))

		if comparison > comparisonEqual {
			slog.Debug(i18n.Msg("checkPackageVersion: downgrade detected, asking for confirmation"))
			var confirm bool
			var confirmErr error
			if confirm, confirmErr = pterm.DefaultInteractiveConfirm.
				WithDefaultValue(false).
				Show(fmt.Sprintf(i18n.Msg("Install version %s (already installed: %s)?"), versionToCheck.Original, latestInstalled.Version)); confirmErr != nil {
				err = fmt.Errorf(i18n.Msg("Installation cancelled: %w"), confirmErr)
				return
			}
			if !confirm {
				err = errors.New(i18n.Msg("Installation cancelled by user"))
				return
			}
		}
	}

	result = versionCheckResult{shouldInstall: true}
	return
}
