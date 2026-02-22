// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package dependency

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
	"github.com/seniorGolang/tg/v3/internal/installer/version"

	"github.com/pterm/pterm"
)

type resolver struct {
	manifestManager managers.ManifestManager
	databaseManager managers.DatabaseManager
}

func NewResolver(manifestManager managers.ManifestManager, databaseManager managers.DatabaseManager) (res managers.DependencyResolver) {
	return &resolver{
		manifestManager: manifestManager,
		databaseManager: databaseManager,
	}
}

// resolveAlias: загружает целевой пакет по alias и объединяет зависимости (целевые + из псевдонима).
func (r *resolver) resolveAlias(ctx context.Context, pkg *models.Package) (effective *models.Package, downloadSource string, targetVersion string, err error) {

	if pkg.Alias == "" {
		effective = pkg
		return
	}

	var parsedURI uri.URI
	if parsedURI, err = uri.New(pkg.Alias); err != nil {
		return nil, "", "", fmt.Errorf(i18n.Msg("Failed to parse alias %s: %w"), pkg.Alias, err)
	}

	source := parsedURI.Source()
	packageName := parsedURI.Package()
	if source == "" || packageName == "" {
		return nil, "", "", errors.New(i18n.Msg("Invalid alias: source and package required"))
	}

	if err = r.ensureManifestLoaded(ctx, source); err != nil {
		return nil, "", "", fmt.Errorf(i18n.Msg("Failed to load manifest for alias %s: %w"), pkg.Alias, err)
	}

	fullName := source + "/" + packageName
	var targetPkg *models.Package
	var targetManifest *models.Manifest
	if targetPkg, targetManifest, err = r.manifestManager.FindPackage(ctx, fullName); err != nil {
		return nil, "", "", fmt.Errorf(i18n.Msg("Failed to find alias target %s: %w"), pkg.Alias, err)
	}

	mergedDeps := mergeDependencies(targetPkg.Dependencies, pkg.Dependencies)

	descr := pkg.Descr
	if descr == "" {
		descr = targetPkg.Descr
	}
	effective = &models.Package{
		Name:         pkg.Name,
		Descr:        descr,
		Downloads:    targetPkg.Downloads,
		Files:        targetPkg.Files,
		Scripts:      targetPkg.Scripts,
		Dependencies: mergedDeps,
	}
	return effective, source, targetManifest.Version, nil
}

func mergeDependencies(base []string, extra []string) (merged []string) {

	seen := make(map[string]bool)
	for _, s := range base {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			merged = append(merged, s)
		}
	}
	for _, s := range extra {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			merged = append(merged, s)
		}
	}
	return
}

func (r *resolver) ResolveDependencies(ctx context.Context, pkg *models.Package) (graph *models.DependencyGraph, err error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	graph = &models.DependencyGraph{
		Nodes: make(map[string]*models.DependencyNode),
		Edges: make([]*models.DependencyEdge, 0),
	}

	pkgToResolve := pkg
	var rootVersion models.Version
	var rootDownloadSource string
	if pkg.Alias != "" {
		var targetVer string
		var effective *models.Package
		if effective, rootDownloadSource, targetVer, err = r.resolveAlias(ctx, pkg); err != nil {
			return
		}
		pkgToResolve = effective
		if targetVer != "" {
			rootVersion, _ = version.Parse(targetVer)
		}
	}

	nodeID := pkgToResolve.Name
	graph.Nodes[nodeID] = &models.DependencyNode{
		Package: pkgToResolve,
		Version: rootVersion,
		ID:      nodeID,
		Source:  rootDownloadSource,
	}

	for _, depStr := range pkgToResolve.Dependencies {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var parsedURI uri.URI
		if parsedURI, err = r.normalizeURI(ctx, depStr); err != nil {
			return nil, fmt.Errorf(i18n.Msg("Failed to parse dependency %s: %w"), depStr, err)
		}

		source := parsedURI.Source()
		packageName := parsedURI.Package()
		versionStr := parsedURI.Version().Original

		if source != "" {
			if err = r.ensureManifestLoaded(ctx, source); err != nil {
				return nil, fmt.Errorf(i18n.Msg("Failed to load manifest from source %s: %w"), source, err)
			}
		}

		var depPkg *models.Package
		var depManifest *models.Manifest

		fullPackageName := packageName
		if source != "" {
			fullPackageName = source + "/" + packageName
		}

		slog.Debug(i18n.Msg("ResolveDependencies: searching for dependency"), slog.String("dependency", packageName), slog.String("source", source), slog.String("packageName", fullPackageName))

		if depPkg, depManifest, err = r.manifestManager.FindPackage(ctx, fullPackageName); err != nil {
			slog.Debug(i18n.Msg("ResolveDependencies: FindPackage failed"), slog.String("packageName", fullPackageName), slog.Any("error", err))
			errMsg := err.Error()
			multipleManifestsMsg := fmt.Sprintf(i18n.Msg("Package %s found in multiple manifests"), packageName)
			if strings.Contains(errMsg, multipleManifestsMsg) || strings.Contains(errMsg, "найден в нескольких манифестах") {
				var resolveErr error
				var selectedPkg *models.Package
				var selectedManifest *models.Manifest
				if selectedPkg, selectedManifest, resolveErr = r.resolveSourceConflict(ctx, packageName, source); resolveErr != nil {
					return nil, fmt.Errorf(i18n.Msg("Failed to resolve dependency %s: %w"), packageName, resolveErr)
				}
				depPkg = selectedPkg
				depManifest = selectedManifest
			} else {
				return nil, fmt.Errorf(i18n.Msg("Failed to find dependency %s: %w"), packageName, err)
			}
		}

		depDownloadSource := source
		var depVersion models.Version
		if depManifest != nil {
			depVersion, _ = version.Parse(depManifest.Version)
		}
		if depPkg.Alias != "" {
			var targetVer string
			var effectiveDep *models.Package
			if effectiveDep, depDownloadSource, targetVer, err = r.resolveAlias(ctx, depPkg); err != nil {
				return nil, fmt.Errorf(i18n.Msg("Failed to resolve alias for dependency %s: %w"), packageName, err)
			}
			depPkg = effectiveDep
			if targetVer != "" {
				depVersion, _ = version.Parse(targetVer)
			}
		}

		depNodeID := fmt.Sprintf("%s/%s", depDownloadSource, depPkg.Name)
		if _, exists := graph.Nodes[depNodeID]; !exists {
			graph.Nodes[depNodeID] = &models.DependencyNode{
				Package: depPkg,
				Version: depVersion,
				ID:      depNodeID,
				Source:  depDownloadSource,
			}
		}

		dep := models.Dependency{
			Source:  depDownloadSource,
			Package: depPkg.Name,
			Version: versionStr,
		}
		graph.Edges = append(graph.Edges, &models.DependencyEdge{
			From:       graph.Nodes[nodeID],
			To:         graph.Nodes[depNodeID],
			Dependency: &dep,
		})
	}

	return
}

// CheckCycles: DFS с рекурсивным стеком, O(V+E).
func (r *resolver) CheckCycles(ctx context.Context, graph *models.DependencyGraph) (err error) {

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var checkCycle func(nodeID string) bool
	checkCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, edge := range graph.Edges {
			if edge.From.ID == nodeID {
				toID := edge.To.ID
				if !visited[toID] {
					if checkCycle(toID) {
						return true
					}
				} else if recStack[toID] {
					return true
				}
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range graph.Nodes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !visited[nodeID] {
			if checkCycle(nodeID) {
				return errors.New(i18n.Msg("Cycle detected in dependencies"))
			}
		}
	}

	return
}

func (r *resolver) SortForInstallation(ctx context.Context, graph *models.DependencyGraph) (packages []*models.Package, err error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err = r.CheckCycles(ctx, graph); err != nil {
		return
	}

	inDegree := make(map[string]int)
	for nodeID := range graph.Nodes {
		inDegree[nodeID] = 0
	}

	for _, edge := range graph.Edges {
		inDegree[edge.To.ID]++
	}

	queue := make([]string, 0)
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := make([]*models.Package, 0)
	for len(queue) > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		nodeID := queue[0]
		queue = queue[1:]

		node := graph.Nodes[nodeID]
		result = append(result, node.Package)

		for _, edge := range graph.Edges {
			if edge.From.ID == nodeID {
				inDegree[edge.To.ID]--
				if inDegree[edge.To.ID] == 0 {
					queue = append(queue, edge.To.ID)
				}
			}
		}
	}

	return result, nil
}

func (r *resolver) CheckCompatibility(ctx context.Context, installed *models.Package, required *models.Dependency) (compatible bool) {

	if required.Version == "" {
		return true
	}

	var err error
	var installation *models.Installation
	if installation, err = r.databaseManager.FindByPackage(ctx, required.Source, installed.Name); err != nil {
		return false
	}

	var installedVersion models.Version
	if installedVersion, err = version.Parse(installation.Version); err != nil {
		return false
	}

	return version.Match(required.Version, installedVersion)
}

func (r *resolver) resolveSourceConflict(ctx context.Context, packageName string, requestedSource string) (pkg *models.Package, manifest *models.Manifest, err error) {

	var allPackages []managers.PackageWithSource
	if allPackages, err = r.manifestManager.FindAllPackages(ctx, packageName); err != nil {
		return nil, nil, fmt.Errorf(i18n.Msg("Failed to get package information: %w"), err)
	}

	if len(allPackages) == 0 {
		return nil, nil, fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
	}

	if len(allPackages) == 1 {
		pkg = allPackages[0].Package
		manifest = allPackages[0].Manifest
		return
	}

	var installed *models.Installation
	if installed, err = r.databaseManager.FindByPackage(ctx, "", packageName); err != nil {
		installed = nil
	}

	if requestedSource != "" {
		normalizedRequestedSource := storage.NormalizeSource(requestedSource)
		for _, pkgWithSource := range allPackages {
			normalizedCurrentSource := storage.NormalizeSource(pkgWithSource.Source)
			if normalizedRequestedSource == normalizedCurrentSource {
				if installed != nil {
					normalizedInstalledSource := storage.NormalizeSource(installed.Source)

					if normalizedInstalledSource == normalizedRequestedSource {
						pkg = pkgWithSource.Package
						manifest = pkgWithSource.Manifest
						return
					}

					var installedVersion models.Version
					if installedVersion, err = version.Parse(installed.Version); err == nil {
						var requestedVersion models.Version
						if requestedVersion, err = version.Parse(pkgWithSource.Manifest.Version); err == nil {
							comparison := version.Compare(installedVersion, requestedVersion)
							if comparison <= 0 {
								pkg = pkgWithSource.Package
								manifest = pkgWithSource.Manifest
								return
							}
							confirmMessage := fmt.Sprintf(i18n.Msg("Replace package %s from source '%s' (version %s) with package from source '%s' (version %s)?"), packageName, installed.Source, installed.Version, pkgWithSource.Source, pkgWithSource.Manifest.Version)
							var confirm bool
							if confirm, err = pterm.DefaultInteractiveConfirm.
								WithDefaultValue(false).
								Show(confirmMessage); err != nil {
								return nil, nil, fmt.Errorf(i18n.Msg("Confirmation cancelled: %w"), err)
							}
							if !confirm {
								return nil, nil, errors.New(i18n.Msg("Package replacement cancelled by user"))
							}
							pkg = pkgWithSource.Package
							manifest = pkgWithSource.Manifest
							return
						}
					}
					confirmMessage := fmt.Sprintf(i18n.Msg("Replace package %s from source '%s' (version %s) with package from source '%s' (version %s)?"), packageName, installed.Source, installed.Version, pkgWithSource.Source, pkgWithSource.Manifest.Version)
					var confirm bool
					if confirm, err = pterm.DefaultInteractiveConfirm.
						WithDefaultValue(false).
						Show(confirmMessage); err != nil {
						return nil, nil, fmt.Errorf(i18n.Msg("Confirmation cancelled: %w"), err)
					}
					if !confirm {
						return nil, nil, errors.New(i18n.Msg("Package replacement cancelled by user"))
					}
				}
				pkg = pkgWithSource.Package
				manifest = pkgWithSource.Manifest
				return
			}
		}
	}

	var conflictMessage strings.Builder
	_, _ = fmt.Fprintf(&conflictMessage, i18n.Msg("Package %s found in multiple sources:")+"\n", packageName)

	if installed != nil {
		_, _ = fmt.Fprintf(&conflictMessage, i18n.Msg("Currently installed: %s (source: %s, version: %s)")+"\n", packageName, installed.Source, installed.Version)
	}

	conflictMessage.WriteString(i18n.Msg("Available sources:") + "\n")

	options := make([]string, 0, len(allPackages))
	packageMap := make(map[string]*managers.PackageWithSource, len(allPackages))

	if requestedSource != "" {
		_, _ = fmt.Fprintf(&conflictMessage, i18n.Msg("Requested source: %s")+"\n", requestedSource)
	}

	for i := range allPackages {
		pkgWithSource := &allPackages[i]
		sourceDisplay := pkgWithSource.Source
		if sourceDisplay == "" {
			sourceDisplay = i18n.Msg("(no source)")
		}

		optionText := fmt.Sprintf("%s - %s v%s", sourceDisplay, packageName, pkgWithSource.Manifest.Version)
		if pkgWithSource.Package.Descr != "" {
			optionText += " - " + pkgWithSource.Package.Descr
		}

		if requestedSource != "" && pkgWithSource.Source == requestedSource {
			optionText += " " + i18n.Msg("(requested)")
		}

		if installed != nil && installed.Source != "" {
			normalizedInstalledSource := strings.TrimSpace(installed.Source)
			normalizedCurrentSource := strings.TrimSpace(pkgWithSource.Source)
			if normalizedInstalledSource == normalizedCurrentSource {
				optionText += " " + i18n.Msg("(currently installed)")
			}
		}

		options = append(options, optionText)
		packageMap[optionText] = pkgWithSource
		_, _ = fmt.Fprintf(&conflictMessage, "  - %s\n", optionText)
	}

	pterm.Warning.Println(conflictMessage.String())

	if installed != nil {
		hasInstalledSource := false
		for _, pkgWithSource := range allPackages {
			if installed.Source != "" && pkgWithSource.Source == installed.Source {
				hasInstalledSource = true
				break
			}
		}

		if !hasInstalledSource {
			replaceMessage := fmt.Sprintf(i18n.Msg("Warning: Package %s is already installed from source '%s' (version %s). Installing from a different source will replace the existing installation."), packageName, installed.Source, installed.Version)
			pterm.Warning.Println(replaceMessage)
		}
	}

	var exists bool
	var selected string
	var selectedPkg *managers.PackageWithSource
	if selected, err = pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(i18n.Msg("Select source for package")); err != nil || selected == "" {
		return nil, nil, errors.New(i18n.Msg("Package source selection cancelled"))
	}

	if selectedPkg, exists = packageMap[selected]; !exists {
		return nil, nil, errors.New(i18n.Msg("Selected package source not found"))
	}

	if installed != nil && installed.Source != "" && selectedPkg.Source != installed.Source {
		confirmMessage := fmt.Sprintf(i18n.Msg("Replace package %s from source '%s' (version %s) with package from source '%s' (version %s)?"), packageName, installed.Source, installed.Version, selectedPkg.Source, selectedPkg.Manifest.Version)
		var confirm bool
		if confirm, err = pterm.DefaultInteractiveConfirm.
			WithDefaultValue(false).
			Show(confirmMessage); err != nil {
			return nil, nil, fmt.Errorf(i18n.Msg("Confirmation cancelled: %w"), err)
		}
		if !confirm {
			return nil, nil, errors.New(i18n.Msg("Package replacement cancelled by user"))
		}
	}

	return selectedPkg.Package, selectedPkg.Manifest, nil
}

func (r *resolver) normalizeURI(ctx context.Context, spec string) (normalizedURI uri.URI, err error) {
	parsedURI, parseErr := uri.New(spec)
	if parseErr == nil {
		normalizedURI = parsedURI
		return
	}

	if !strings.Contains(parseErr.Error(), "URL must have a scheme") {
		return uri.URI{}, parseErr
	}

	parts := strings.Split(spec, "@")
	packageName := parts[0]
	versionStr := ""
	if len(parts) > 1 {
		versionStr = parts[1]
	}

	var packages []managers.PackageWithSource
	if packages, err = r.manifestManager.FindAllPackages(ctx, packageName); err != nil {
		return uri.URI{}, fmt.Errorf(i18n.Msg("Package %s not found: %w"), packageName, err)
	}

	if len(packages) == 0 {
		return uri.URI{}, fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
	}

	source := packages[0].Source
	if source == "" {
		return uri.URI{}, fmt.Errorf(i18n.Msg("Package %s found but source is empty"), packageName)
	}

	normalizedSpec := source + ":" + packageName
	if versionStr != "" {
		normalizedSpec += "@" + versionStr
	}

	if normalizedURI, err = uri.New(normalizedSpec); err != nil {
		return uri.URI{}, fmt.Errorf(i18n.Msg("Failed to parse normalized URI %s: %w"), normalizedSpec, err)
	}

	return
}

func (r *resolver) ensureManifestLoaded(ctx context.Context, source string) (err error) {

	slog.Debug(i18n.Msg("Ensuring manifest is loaded"), slog.String("source", source))

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(source); err == nil {
		if parsedURL.Host == "github.com" || strings.HasSuffix(parsedURL.Host, ".github.com") {
			slog.Debug(i18n.Msg("Updating GitHub manifest"), slog.String("source", source))
			if err = r.manifestManager.UpdateManifest(ctx, source, false); err != nil {
				slog.Error(i18n.Msg("Failed to update GitHub manifest"), slog.String("source", source), slog.Any("error", err))
				return
			}
			if err = r.manifestManager.ReloadIndex(ctx); err != nil {
				slog.Error(i18n.Msg("Failed to reload index after updating manifest"), slog.String("source", source), slog.Any("error", err))
				return
			}
			slog.Debug(i18n.Msg("Successfully updated and reloaded GitHub manifest"), slog.String("source", source))
			return
		}
	}

	var parsedURI uri.URI
	if parsedURI, err = uri.New(source); err != nil {
		return fmt.Errorf("failed to parse source URL: %w", err)
	}

	var manifestURL string
	if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
		return fmt.Errorf("failed to build manifest URL: %w", err)
	}

	slog.Debug(i18n.Msg("Loading manifest cascade"), slog.String("source", source), slog.String("manifestURL", manifestURL))
	if _, err = r.manifestManager.LoadManifestCascade(ctx, manifestURL, source, false); err != nil {
		slog.Error(i18n.Msg("Failed to load manifest cascade"), slog.String("source", source), slog.String("manifestURL", manifestURL), slog.Any("error", err))
		return
	}

	if err = r.manifestManager.ReloadIndex(ctx); err != nil {
		slog.Error(i18n.Msg("Failed to reload index after loading manifest"), slog.String("source", source), slog.Any("error", err))
		return
	}

	slog.Debug(i18n.Msg("Successfully loaded manifest"), slog.String("source", source), slog.String("manifestURL", manifestURL))
	return
}
