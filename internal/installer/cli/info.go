// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"

	"github.com/pterm/pterm"
)

// HandleInfo обрабатывает команду info.
func (inst *Installer) HandleInfo(ctx context.Context, args []string) (err error) {

	var packageName string
	if len(args) == 0 {
		if packageName, err = inst.selectPackageInteractive(ctx); err != nil {
			return
		}
	} else {
		packageName = args[0]
	}

	var pkg *models.Package
	var manifest *models.Manifest
	if pkg, manifest, err = inst.manifestManager.FindPackage(ctx, packageName); err != nil {
		return fmt.Errorf(i18n.Msg("Package not found: %w"), err)
	}

	header := []string{i18n.Msg("Property"), i18n.Msg("Value")}
	rows := [][]string{
		{i18n.Msg("Package"), pkg.Name},
		{i18n.Msg("Version"), manifest.Version},
		{i18n.Msg("Description"), pkg.Descr},
	}

	if err = renderMarkdownTable(header, rows); err != nil {
		return
	}

	return
}

// selectPackageInteractive предлагает интерактивный выбор пакета из установленных.
func (inst *Installer) selectPackageInteractive(ctx context.Context) (packageName string, err error) {

	var installations []models.Installation
	if installations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to get list of installed packages: %w"), err)
	}

	if len(installations) == 0 {
		return "", errors.New(i18n.Msg("No packages installed"))
	}

	options := make([]string, 0, len(installations))
	packageMap := make(map[string]string, len(installations))

	for i := range installations {
		installation := &installations[i]

		fullPackageName := installation.Package
		if installation.Source != "" {
			fullPackageName = installation.Source + packagePathSeparator + installation.Package
		}

		optionText := installation.Package
		if installation.Version != "" {
			optionText = fmt.Sprintf("%s [%s]", installation.Package, installation.Version)
		}
		if installation.Descr != "" {
			if installation.Version != "" {
				optionText = fmt.Sprintf("%s [%s] - %s", installation.Package, installation.Version, installation.Descr)
			} else {
				optionText = fmt.Sprintf("%s - %s", installation.Package, installation.Descr)
			}
		}

		options = append(options, optionText)
		packageMap[optionText] = fullPackageName
	}

	sort.Strings(options)

	var selected string
	selected, err = pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(i18n.Msg("Select package to view information"))

	if err != nil || selected == "" {
		return "", errors.New(i18n.Msg("Package selection cancelled"))
	}

	var exists bool
	packageName, exists = packageMap[selected]
	if !exists {
		return "", errors.New(i18n.Msg("Selected package not found"))
	}

	return
}
