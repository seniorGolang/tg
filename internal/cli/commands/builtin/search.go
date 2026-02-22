// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"errors"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/cli"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"

	"github.com/pterm/pterm"
)

func findPackageWithSelection(ctx types.CommandContext, inst *cli.Installer, packageName string, installation *models.Installation) (pkg *models.Package, manifest *models.Manifest, err error) {

	cmdCtx := context.Background()

	if installation != nil && installation.Source != "" {
		fullPackageName := installation.Source + pathSeparator + installation.Package
		if pkg, manifest, err = inst.ManifestManager().FindPackage(cmdCtx, fullPackageName); err == nil {
			return
		}
	}

	if pkg, manifest, err = inst.ManifestManager().FindPackage(cmdCtx, packageName); err == nil {
		return
	}

	errMsg := err.Error()
	multipleManifestsMsgEn := packageFoundInMultipleManifests + packageName + foundInMultipleManifestsSuffix
	multipleManifestsMsgRu := i18n.Msg("found in multiple manifests")
	if strings.Contains(errMsg, multipleManifestsMsgEn) || strings.Contains(errMsg, multipleManifestsMsgRu) {
		return selectPackageFromMultiple(ctx, inst, packageName, cmdCtx)
	}

	return
}

func findPackageInCatalog(ctx types.CommandContext, inst *cli.Installer, packageName string, version string) (pkg *models.Package, manifest *models.Manifest, err error) {

	cmdCtx := context.Background()

	var allPackages []managers.PackageWithSource
	if allPackages, err = inst.ManifestManager().FindAllPackages(cmdCtx, packageName); err != nil {
		return
	}

	if len(allPackages) == 0 {
		notFoundMsg := i18n.Msg("Package ") + packageName + i18n.Msg(" not found")
		return nil, nil, errors.New(notFoundMsg)
	}

	if version == "" && len(allPackages) == 1 {
		return allPackages[0].Package, allPackages[0].Manifest, nil
	}

	if version != "" {
		normalizedVersion := strings.TrimPrefix(version, versionPrefixV)
		for i := range allPackages {
			manifestNormalizedVersion := strings.TrimPrefix(allPackages[i].Manifest.Version, versionPrefixV)
			if manifestNormalizedVersion == normalizedVersion {
				return allPackages[i].Package, allPackages[i].Manifest, nil
			}
		}
		notFoundMsg := i18n.Msg("Package ") + packageName + i18n.Msg(" version ") + version + i18n.Msg(" not found")
		return nil, nil, errors.New(notFoundMsg)
	}

	if len(allPackages) > 1 {
		return selectPackageFromMultiple(ctx, inst, packageName, cmdCtx)
	}

	return allPackages[0].Package, allPackages[0].Manifest, nil
}

func selectPackageFromMultiple(ctx types.CommandContext, inst *cli.Installer, packageName string, cmdCtx context.Context) (pkg *models.Package, manifest *models.Manifest, err error) {

	var findErr error
	var allPackages []managers.PackageWithSource
	if allPackages, findErr = inst.ManifestManager().FindAllPackages(cmdCtx, packageName); findErr != nil {
		failedMsg := i18n.Msg("Failed to get package information: ") + findErr.Error()
		return nil, nil, errors.New(failedMsg)
	}

	if len(allPackages) == 0 {
		notFoundMsg := i18n.Msg("Package ") + packageName + i18n.Msg(" not found")
		return nil, nil, errors.New(notFoundMsg)
	}

	if len(allPackages) == 1 {
		return allPackages[0].Package, allPackages[0].Manifest, nil
	}

	options := make([]string, 0, len(allPackages))
	packageMap := make(map[string]*managers.PackageWithSource, len(allPackages))

	for i := range allPackages {
		pkgWithSource := &allPackages[i]
		optionText := formatPackageOption(pkgWithSource)
		options = append(options, optionText)
		packageMap[optionText] = pkgWithSource
	}

	selected, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(i18n.Msg("Select package version to view documentation"))

	if selected == "" {
		return nil, nil, errors.New(i18n.Msg("Package selection cancelled"))
	}

	var exists bool
	var selectedPkg *managers.PackageWithSource
	selectedPkg, exists = packageMap[selected]
	if !exists {
		return nil, nil, errors.New(i18n.Msg("Selected package not found"))
	}

	return selectedPkg.Package, selectedPkg.Manifest, nil
}

func formatPackageOption(pkgWithSource *managers.PackageWithSource) (optionText string) {

	optionText = pkgWithSource.Package.Name
	if pkgWithSource.Source != "" {
		optionText = pkgWithSource.Package.Name + packageSourcePrefix + pkgWithSource.Source + packageSourceSuffix
	}
	if pkgWithSource.Manifest.Version != "" {
		optionText += docVersionSeparator + pkgWithSource.Manifest.Version
	}
	if pkgWithSource.Package.Descr != "" {
		optionText += docDescriptionSeparator + pkgWithSource.Package.Descr
	}
	return
}

func findInstallation(installations []models.Installation, pluginName string, version string) (foundInstallation *models.Installation) {

	normalizedVersion := strings.TrimPrefix(version, versionPrefixV)

	for i := range installations {
		if installations[i].Package == pluginName {
			if version == "" {
				return &installations[i]
			}
			instNormalizedVersion := strings.TrimPrefix(installations[i].Version, versionPrefixV)
			if instNormalizedVersion == normalizedVersion {
				return &installations[i]
			}
		}
	}

	return
}

func parsePluginArg(arg string) (pluginName string, version string) {

	idx := strings.Index(arg, versionSeparator)
	if idx > 0 {
		pluginName = arg[:idx]
		version = arg[idx+len(versionSeparator):]
	} else {
		pluginName = arg
	}
	return
}
