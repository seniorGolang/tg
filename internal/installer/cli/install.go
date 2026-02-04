// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/installation"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

func (inst *Installer) HandleInstall(ctx context.Context, args []string, version string, force bool, dryRun bool, verbose bool) (err error) {

	if len(args) == 0 {
		err = errors.New(i18n.Msg("Package not specified for installation"))
		return
	}

	if dryRun {
		for _, packageSpec := range args {
			var parsedURI uri.URI
			if parsedURI, err = inst.normalizeURI(ctx, packageSpec); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to parse package specification: %w"), err)
				return
			}

			source := parsedURI.Source()
			packageName := parsedURI.Package()
			versionStr := parsedURI.Version().Original

			if version != "" {
				versionStr = version
			}

			switch {
			case source == "":
				var pkg *models.Package
				var manifest *models.Manifest
				pkg, manifest, err = inst.manifestManager.FindPackage(ctx, packageName)
				if err != nil {
					if isMultipleManifestsError(err) {
						var selectedSource string
						pkg, manifest, selectedSource, err = inst.resolvePackageWithSourceSelection(ctx, packageName)
						_ = selectedSource
					}
				}
				if err != nil {
					err = fmt.Errorf(i18n.Msg("Package not found: %w"), err)
					return
				}
				fmt.Printf(i18n.Msg("Package will be installed: %s@%s")+"\n", pkg.Name, manifest.Version)
			case packageName == "":
				fmt.Printf(i18n.Msg("Interactive package selection from source will be performed: %s")+"\n", source)
			default:
				fmt.Printf(i18n.Msg("Package will be installed: %s/%s@%s")+"\n", source, packageName, versionStr)
			}
		}
		return
	}

	for _, packageSpec := range args {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		if err = inst.installSinglePackage(ctx, packageSpec, version, force, verbose); err != nil {
			err = fmt.Errorf(i18n.Msg("Installation error from %s: %w"), packageSpec, err)
			return
		}
	}

	return
}

func (inst *Installer) normalizeURI(ctx context.Context, spec string) (normalizedURI uri.URI, err error) {

	parsedURI, parseErr := uri.New(spec)
	if parseErr == nil {
		normalizedURI = parsedURI
		return
	}

	if !strings.Contains(parseErr.Error(), "URL must have a scheme") {
		err = parseErr
		return
	}

	parts := strings.Split(spec, "@")
	packageName := parts[0]
	versionStr := ""
	if len(parts) > 1 {
		versionStr = parts[1]
	}

	var packages []managers.PackageWithSource
	if packages, err = inst.manifestManager.FindAllPackages(ctx, packageName); err != nil {
		err = fmt.Errorf(i18n.Msg("Package %s not found: %w"), packageName, err)
		return
	}

	if len(packages) == 0 {
		err = fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
		return
	}

	source := packages[0].Source
	if source == "" {
		err = fmt.Errorf(i18n.Msg("Package %s found but source is empty"), packageName)
		return
	}

	normalizedSpec := source + ":" + packageName
	if versionStr != "" {
		normalizedSpec += "@" + versionStr
	}

	if normalizedURI, err = uri.New(normalizedSpec); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse normalized URI %s: %w"), normalizedSpec, err)
		return
	}

	return
}

func (inst *Installer) installSinglePackage(ctx context.Context, packageSpec string, versionOpt string, force bool, verbose bool) (err error) {

	var parsedURI uri.URI
	if parsedURI, err = inst.normalizeURI(ctx, packageSpec); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse package specification: %w"), err)
		return
	}

	source := parsedURI.Source()
	packageName := parsedURI.Package()
	versionStr := parsedURI.Version().Original

	if versionOpt != "" {
		versionStr = versionOpt
	}

	if source == "" {
		var pkg *models.Package
		var manifest *models.Manifest
		pkg, manifest, err = inst.manifestManager.FindPackage(ctx, packageName)
		if err != nil {
			if isMultipleManifestsError(err) {
				var selectedSource string
				pkg, manifest, selectedSource, err = inst.resolvePackageWithSourceSelection(ctx, packageName)
				_ = selectedSource
			}
		}
		if err != nil {
			err = fmt.Errorf(i18n.Msg("Package not found: %w"), err)
			return
		}

		var v models.Version
		if v, err = version.Parse(manifest.Version); err != nil {
			err = fmt.Errorf(i18n.Msg("Invalid version format: %w"), err)
			return
		}

		if versionStr != "" {
			var requestedVersion models.Version
			if requestedVersion, err = version.Parse(versionStr); err != nil {
				err = fmt.Errorf(i18n.Msg("Invalid requested version format: %w"), err)
				return
			}
			if version.Compare(v, requestedVersion) != 0 {
				err = fmt.Errorf(i18n.Msg("Requested version %s does not match manifest version %s"), versionStr, manifest.Version)
				return
			}
		}

		ctx = context.WithValue(ctx, ContextKeyForce, force)
		// При --force не удаляем установку заранее: при ошибке старая остаётся; ID = package@version, RecordInstallation обновит запись.
		return inst.installationManager.Install(ctx, pkg, v)
	}

	if packageName == "" {
		return inst.handleInstallFromSource(ctx, source, versionStr, force)
	}

	return inst.handleInstallPackageFromSource(ctx, source, packageName, versionStr, force)
}

func (inst *Installer) handleInstallFromSource(ctx context.Context, source string, versionStr string, force bool) (err error) {

	slog.Debug(i18n.Msg("Handling install from source"), slog.String("source", source), slog.String("version", versionStr), slog.Bool("force", force))

	var parsedURI uri.URI
	if parsedURI, err = uri.New(source); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse source URL: %w"), err)
		return
	}

	var manifestURL string
	if manifestURL, err = parsedURI.ManifestURL(ctx, versionStr); err != nil {
		slog.Error(i18n.Msg("Failed to get manifest"), slog.String("source", source), slog.String("version", versionStr), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to get manifest: %w"), err)
		return
	}

	slog.Debug(i18n.Msg("Loading manifest cascade"), slog.String("source", source), slog.String("manifestURL", manifestURL))
	var loadedSources map[string]bool
	if loadedSources, err = inst.manifestManager.LoadManifestCascade(ctx, manifestURL, source, false); err != nil {
		slog.Debug(i18n.Msg("Failed to load manifest cascade, trying JSON fallback"), slog.String("source", source), slog.String("manifestURL", manifestURL), slog.Any("error", err))
		if strings.Contains(manifestURL, storage.ReleasesDownloadPath) && strings.HasSuffix(manifestURL, storage.ManifestFileName) {
			jsonURL := strings.TrimSuffix(manifestURL, storage.ManifestFileName) + ".json"
			slog.Debug(i18n.Msg("Trying JSON manifest"), slog.String("source", source), slog.String("jsonURL", jsonURL))
			var jsonErr error
			if loadedSources, jsonErr = inst.manifestManager.LoadManifestCascade(ctx, jsonURL, source, false); jsonErr == nil {
				if err = inst.manifestManager.ReloadIndex(ctx); err != nil {
					slog.Error(i18n.Msg("Failed to reload index after loading manifest cascade"), slog.String("source", source), slog.Any("error", err))
					err = fmt.Errorf(i18n.Msg("Failed to reload index: %w"), err)
					return
				}
				return inst.processAllPackagesFromManifests(ctx, versionStr, force, source, loadedSources)
			}
			slog.Debug(i18n.Msg("JSON manifest fallback also failed"), slog.String("source", source), slog.String("jsonURL", jsonURL), slog.Any("error", jsonErr))
		}
		slog.Error(i18n.Msg("Failed to load manifest"), slog.String("source", source), slog.String("manifestURL", manifestURL), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
		return
	}

	if err = inst.manifestManager.ReloadIndex(ctx); err != nil {
		slog.Error(i18n.Msg("Failed to reload index after loading manifest cascade"), slog.String("source", source), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to reload index: %w"), err)
		return
	}

	return inst.processAllPackagesFromManifests(ctx, versionStr, force, source, loadedSources)
}

func (inst *Installer) processAllPackagesFromManifests(ctx context.Context, versionStr string, force bool, source string, loadedSources map[string]bool) (err error) {

	var allPackages []models.Package
	if allPackages, err = inst.manifestManager.ListPackagesFromSources(ctx, loadedSources); err != nil {
		slog.Error(i18n.Msg("Failed to list packages from manifests"), slog.String("source", source), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to list packages: %w"), err)
		return
	}

	if len(allPackages) == 0 {
		slog.Warn(i18n.Msg("No packages in loaded manifests"), slog.String("source", source))
		return
	}

	var mainManifest *models.Manifest
	var parsedURI uri.URI
	if parsedURI, err = uri.New(source); err == nil {
		var manifestURL string
		if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err == nil {
			if mainManifest, err = inst.manifestManager.LoadManifest(ctx, manifestURL); err != nil {
				slog.Debug(i18n.Msg("Failed to load main manifest for version check"), slog.Any("error", err))
			}
		}
	}

	if versionStr != "" && mainManifest != nil {
		var manifestVersion models.Version
		if manifestVersion, err = version.Parse(mainManifest.Version); err == nil {
			var requestedVersion models.Version
			if requestedVersion, err = version.Parse(versionStr); err == nil {
				if version.Compare(manifestVersion, requestedVersion) != 0 {
					err = fmt.Errorf(i18n.Msg("Manifest version %s does not match requested %s"), mainManifest.Version, versionStr)
					return
				}
			}
		}
	}

	ctx = context.WithValue(ctx, ContextKeyForce, force)
	ctx = context.WithValue(ctx, ContextKeySource, source)
	// Один пакет может быть в нескольких манифестах; selectedSourceForConflict — выбранный пользователем источник, чтобы не спрашивать повторно.
	var selectedSourceForConflict string

	getPackageVersion := func(pkg *models.Package) (pkgVersion models.Version, err error) {
		if versionStr != "" {
			if pkgVersion, err = version.Parse(versionStr); err != nil {
				err = fmt.Errorf(i18n.Msg("Invalid version format: %w"), err)
				return
			}
			return
		}

		var pkgManifest *models.Manifest
		if pkgManifest, err = inst.getManifestForPackage(ctx, pkg, &selectedSourceForConflict); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to find manifest for package %s: %w"), pkg.Name, err)
			return
		}
		if pkgVersion, err = version.Parse(pkgManifest.Version); err != nil {
			err = fmt.Errorf(i18n.Msg("Invalid manifest version format for package %s: %w"), pkg.Name, err)
			return
		}
		return
	}

	if len(allPackages) == 1 {
		pkg := allPackages[0]
		var pkgVersion models.Version
		if pkgVersion, err = getPackageVersion(&pkg); err != nil {
			return
		}
		return inst.installationManager.Install(ctx, &pkg, pkgVersion)
	}

	var selectedPackages []*models.Package
	if selectedPackages, err = inst.selectPackagesInteractively(ctx, allPackages, versionStr, &selectedSourceForConflict); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to select packages: %w"), err)
		return
	}

	if len(selectedPackages) == 0 {
		err = errors.New(i18n.Msg("No packages selected"))
		return
	}

	totalPackages := len(selectedPackages)
	if totalPackages > 1 {
		pterm.Info.Printf(i18n.Msg("Installing %d packages...")+"\n", totalPackages)
	}

	var batchCollector *batchTreeCollector
	if totalPackages > 1 {
		batchCollector = &batchTreeCollector{}
		ctx = context.WithValue(ctx, contextkeys.TreeCollector, batchCollector)
		ctx = context.WithValue(ctx, contextkeys.SessionInstalledIDs, &installation.SessionInstalledSet{
			IDs: make(map[string]struct{}),
		})
	}

	for _, pkg := range selectedPackages {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var pkgVersion models.Version
		if pkgVersion, err = getPackageVersion(pkg); err != nil {
			if totalPackages > 1 {
				pterm.Error.Printf(" %s\n", err.Error())
			} else {
				return
			}
			continue
		}

		if installErr := inst.installationManager.Install(ctx, pkg, pkgVersion); installErr != nil {
			if totalPackages > 1 {
				pterm.Error.Printf(" %s\n", installErr.Error())
			} else {
				err = installErr
				return
			}
		}
	}

	if batchCollector != nil && len(batchCollector.trees) > 0 {
		pterm.Success.Println(i18n.Msg("Packages installed:"))
		inst.renderCombinedDependencyTree(batchCollector.trees)
	}

	return
}

type batchTreeCollector struct {
	trees []struct {
		source      string
		rootDisplay string
		deps        []installation.PackageDisplay
	}
}

func (c *batchTreeCollector) AddTree(source string, rootDisplay string, deps []installation.PackageDisplay) {
	c.trees = append(c.trees, struct {
		source      string
		rootDisplay string
		deps        []installation.PackageDisplay
	}{source: source, rootDisplay: rootDisplay, deps: deps})
}

var _ installation.TreeCollector = (*batchTreeCollector)(nil)

func installStatusPriority(status string) (p int) {
	switch status {
	case installation.InstallStatusNew:
		return 2
	case installation.InstallStatusUpdated:
		return 1
	default:
		return 0
	}
}

func (inst *Installer) renderCombinedDependencyTree(trees []struct {
	source      string
	rootDisplay string
	deps        []installation.PackageDisplay
}) {
	if len(trees) == 0 {
		return
	}
	rootSource := trees[0].source

	mergedDeps := make(map[string]installation.PackageDisplay)
	for _, t := range trees {
		for _, d := range t.deps {
			if existing, ok := mergedDeps[d.Name]; !ok || installStatusPriority(d.Status) > installStatusPriority(existing.Status) {
				mergedDeps[d.Name] = d
			}
		}
	}

	depNames := make([]string, 0, len(mergedDeps))
	for name := range mergedDeps {
		depNames = append(depNames, name)
	}
	sort.Strings(depNames)
	dependencyNodes := make([]pterm.TreeNode, 0, len(depNames))
	for _, name := range depNames {
		dependencyNodes = append(dependencyNodes, pterm.TreeNode{Text: mergedDeps[name].Display})
	}

	packageNodes := make([]pterm.TreeNode, 0, len(trees))
	for _, t := range trees {
		packageNodes = append(packageNodes, pterm.TreeNode{Text: t.rootDisplay})
	}

	depsLabel := i18n.Msg("Dependencies")
	packagesLabel := i18n.Msg("Packages")
	rootChildren := []pterm.TreeNode{
		{Text: depsLabel, Children: dependencyNodes},
		{Text: packagesLabel, Children: packageNodes},
	}
	root := pterm.TreeNode{Text: rootSource, Children: rootChildren}
	_ = pterm.DefaultTree.WithRoot(root).Render()
}

func (inst *Installer) handleInstallPackageFromSource(ctx context.Context, source string, packageName string, versionStr string, force bool) (err error) {

	slog.Debug(i18n.Msg("Handling install package from source"), slog.String("source", source), slog.String("package", packageName), slog.String("version", versionStr), slog.Bool("force", force))

	var parsedURI uri.URI
	if parsedURI, err = uri.New(source); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse source URL: %w"), err)
		return
	}

	var manifestURL string
	if manifestURL, err = parsedURI.ManifestURL(ctx, versionStr); err != nil {
		slog.Error(i18n.Msg("Failed to get manifest"), slog.String("source", source), slog.String("package", packageName), slog.String("version", versionStr), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to get manifest: %w"), err)
		return
	}

	slog.Debug(i18n.Msg("Loading manifest cascade for package"), slog.String("source", source), slog.String("package", packageName), slog.String("manifestURL", manifestURL))
	if _, err = inst.manifestManager.LoadManifestCascade(ctx, manifestURL, source, false); err != nil {
		slog.Error(i18n.Msg("Failed to load manifest cascade"), slog.String("source", source), slog.String("package", packageName), slog.String("manifestURL", manifestURL), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
		return
	}

	var manifest *models.Manifest
	if manifest, err = inst.manifestManager.LoadManifest(ctx, manifestURL); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
		return
	}

	if versionStr != "" {
		var manifestVersion models.Version
		if manifestVersion, err = version.Parse(manifest.Version); err != nil {
			err = fmt.Errorf(i18n.Msg("Invalid manifest version format: %w"), err)
			return
		}
		var requestedVersion models.Version
		if requestedVersion, err = version.Parse(versionStr); err != nil {
			err = fmt.Errorf(i18n.Msg("Invalid requested version format: %w"), err)
			return
		}
		if version.Compare(manifestVersion, requestedVersion) != 0 {
			err = fmt.Errorf(i18n.Msg("Manifest version %s does not match requested %s"), manifest.Version, versionStr)
			return
		}
	}

	var pkg *models.Package
	for i := range manifest.Packages {
		if manifest.Packages[i].Name == packageName {
			pkg = &manifest.Packages[i]
			break
		}
	}

	if pkg == nil {
		err = fmt.Errorf(i18n.Msg("Package %s not found in manifest"), packageName)
		return
	}

	var v models.Version
	if v, err = version.Parse(manifest.Version); err != nil {
		err = fmt.Errorf(i18n.Msg("Invalid version format: %w"), err)
		return
	}

	ctx = context.WithValue(ctx, ContextKeyForce, force)
	ctx = context.WithValue(ctx, ContextKeySource, source)
	// При --force не удаляем установку заранее: при ошибке старая остаётся; ID = package@version, RecordInstallation обновит запись.
	return inst.installationManager.Install(ctx, pkg, v)
}

func isMultipleManifestsError(err error) bool {

	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "found in multiple manifests") || strings.Contains(msg, "найден в нескольких манифестах")
}

// getManifestForPackage: для пакета с Alias ищет по целевому source/package без запроса выбора источника.
func (inst *Installer) getManifestForPackage(ctx context.Context, pkg *models.Package, selectedSourceForConflict *string) (manifest *models.Manifest, err error) {

	if pkg.Alias != "" {
		var parsedURI uri.URI
		if parsedURI, err = uri.New(pkg.Alias); err != nil {
			return nil, err
		}
		src := parsedURI.Source()
		pkgName := parsedURI.Package()
		if src == "" || pkgName == "" {
			return nil, fmt.Errorf("%s: %s", i18n.Msg("Invalid alias"), pkg.Alias)
		}
		fullName := src + "/" + pkgName
		_, manifest, err = inst.manifestManager.FindPackage(ctx, fullName)
		return manifest, err
	}

	_, manifest, err = inst.manifestManager.FindPackage(ctx, pkg.Name)
	if err != nil && isMultipleManifestsError(err) {
		if selectedSourceForConflict != nil && *selectedSourceForConflict != "" {
			fullName := strings.TrimSuffix(*selectedSourceForConflict, "/") + "/" + pkg.Name
			_, manifest, err = inst.manifestManager.FindPackage(ctx, fullName)
		} else if selectedSourceForConflict != nil {
			var resPkg *models.Package
			var resSource string
			resPkg, manifest, resSource, err = inst.resolvePackageWithSourceSelection(ctx, pkg.Name)
			if err == nil {
				*selectedSourceForConflict = resSource
			}
			_ = resPkg
		}
	}
	return manifest, err
}

func (inst *Installer) resolvePackageWithSourceSelection(ctx context.Context, packageName string) (pkg *models.Package, manifest *models.Manifest, selectedSource string, err error) {

	allPackages, listErr := inst.manifestManager.FindAllPackages(ctx, packageName)
	if listErr != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get package information: %w"), listErr)
		return
	}

	if len(allPackages) == 0 {
		err = fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
		return
	}

	if len(allPackages) == 1 {
		pkg = allPackages[0].Package
		manifest = allPackages[0].Manifest
		selectedSource = allPackages[0].Source
		return
	}

	options := make([]string, 0, len(allPackages))
	packageMap := make(map[string]*managers.PackageWithSource, len(allPackages))
	for i := range allPackages {
		pws := &allPackages[i]
		sourceDisplay := pws.Source
		if sourceDisplay == "" {
			sourceDisplay = i18n.Msg("(no source)")
		}
		optionText := fmt.Sprintf("%s - %s v%s", sourceDisplay, packageName, pws.Manifest.Version)
		if pws.Package.Descr != "" {
			optionText += " - " + pws.Package.Descr
		}
		options = append(options, optionText)
		packageMap[optionText] = pws
	}

	pterm.Warning.Printf(i18n.Msg("Package %s found in multiple sources:")+"\n", packageName)
	var selected string
	if selected, err = pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(i18n.Msg("Select source for package")); err != nil || selected == "" {
		err = errors.New(i18n.Msg("Package source selection cancelled"))
		return
	}

	var ok bool
	var selectedPws *managers.PackageWithSource
	if selectedPws, ok = packageMap[selected]; !ok {
		err = errors.New(i18n.Msg("Selected package source not found"))
		return
	}

	pkg = selectedPws.Package
	manifest = selectedPws.Manifest
	selectedSource = selectedPws.Source
	return
}

func (inst *Installer) selectPackagesInteractively(ctx context.Context, packages []models.Package, version string, selectedSourceForConflict *string) (selectedPackages []*models.Package, err error) {

	if len(packages) == 0 {
		err = errors.New(i18n.Msg("No packages available for selection"))
		return
	}

	options := make([]string, 0, len(packages))
	packageMap := make(map[string]*models.Package, len(packages))

	for i := range packages {
		pkg := &packages[i]

		pkgVersion := version
		if pkgVersion == "" {
			var pkgManifest *models.Manifest
			var versionErr error
			if pkgManifest, versionErr = inst.getManifestForPackage(ctx, pkg, selectedSourceForConflict); versionErr == nil && pkgManifest != nil {
				pkgVersion = pkgManifest.Version
			}
		}

		optionText := pkg.Name
		if pkgVersion != "" {
			optionText = fmt.Sprintf("%s [%s]", pkg.Name, pkgVersion)
		}
		if pkg.Descr != "" {
			if pkgVersion != "" {
				optionText = fmt.Sprintf("%s [%s] - %s", pkg.Name, pkgVersion, pkg.Descr)
			} else {
				optionText = fmt.Sprintf("%s - %s", pkg.Name, pkg.Descr)
			}
		}
		options = append(options, optionText)
		packageMap[optionText] = pkg
	}

	prompt := i18n.Msg("Select packages to install")
	selectedOptions, err := pterm.DefaultInteractiveMultiselect.
		WithOptions(options).
		WithDefaultOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(prompt)

	if err != nil {
		err = fmt.Errorf(i18n.Msg("Package selection cancelled or failed: %w"), err)
		return
	}

	if len(selectedOptions) == 0 {
		err = errors.New(i18n.Msg("No packages selected"))
		return
	}

	selectedPackages = make([]*models.Package, 0, len(selectedOptions))
	for _, option := range selectedOptions {
		var pkg *models.Package
		var ok bool
		if pkg, ok = packageMap[option]; !ok {
			err = fmt.Errorf(i18n.Msg("Selected package not found: %s"), option)
			return
		}
		selectedPackages = append(selectedPackages, pkg)
	}

	return
}
