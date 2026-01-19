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

// resolver реализует DependencyResolver.
type resolver struct {
	manifestManager managers.ManifestManager
	databaseManager managers.DatabaseManager
}

func NewResolver(manifestManager managers.ManifestManager, databaseManager managers.DatabaseManager) managers.DependencyResolver {
	return &resolver{
		manifestManager: manifestManager,
		databaseManager: databaseManager,
	}
}

// ResolveDependencies разрешает зависимости пакета.
func (r *resolver) ResolveDependencies(ctx context.Context, pkg *models.Package) (graph *models.DependencyGraph, err error) {

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	graph = &models.DependencyGraph{
		Nodes: make(map[string]*models.DependencyNode),
		Edges: make([]*models.DependencyEdge, 0),
	}

	nodeID := pkg.Name
	graph.Nodes[nodeID] = &models.DependencyNode{
		Package: pkg,
		ID:      nodeID,
	}

	for _, depStr := range pkg.Dependencies {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		// Нормализуем URI зависимости (преобразуем packageName@version в source:packageName@version)
		var parsedURI uri.URI
		if parsedURI, err = r.normalizeURI(ctx, depStr); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to parse dependency %s: %w"), depStr, err)
			return
		}

		source := parsedURI.Source()
		packageName := parsedURI.Package()
		versionStr := parsedURI.Version().Original

		// Если указан source, пытаемся загрузить манифест из него, если он еще не загружен
		if source != "" {
			if err = r.ensureManifestLoaded(ctx, source); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to load manifest from source %s: %w"), source, err)
				return
			}
		}

		// Пытаемся найти пакет, используя source из зависимости, если указан
		var depPkg *models.Package
		var depManifest *models.Manifest

		fullPackageName := packageName
		if source != "" {
			fullPackageName = source + "/" + packageName
		}

		slog.Debug(i18n.Msg("ResolveDependencies: searching for dependency"), slog.String("dependency", packageName), slog.String("source", source), slog.String("packageName", fullPackageName))

		if depPkg, depManifest, err = r.manifestManager.FindPackage(ctx, fullPackageName); err != nil {
			slog.Debug(i18n.Msg("ResolveDependencies: FindPackage failed"), slog.String("packageName", fullPackageName), slog.Any("error", err))
			// Проверяем, является ли ошибка конфликтом источников
			errMsg := err.Error()
			multipleManifestsMsg := fmt.Sprintf(i18n.Msg("Package %s found in multiple manifests"), packageName)
			if strings.Contains(errMsg, multipleManifestsMsg) || strings.Contains(errMsg, "найден в нескольких манифестах") {
				// Разрешаем конфликт интерактивно
				var selectedPkg *models.Package
				var selectedManifest *models.Manifest
				var resolveErr error
				if selectedPkg, selectedManifest, resolveErr = r.resolveSourceConflict(ctx, packageName, source); resolveErr != nil {
					err = fmt.Errorf(i18n.Msg("Failed to resolve dependency %s: %w"), packageName, resolveErr)
					return
				}
				depPkg = selectedPkg
				depManifest = selectedManifest
			} else {
				err = fmt.Errorf(i18n.Msg("Failed to find dependency %s: %w"), packageName, err)
				return
			}
		}

		depNodeID := fmt.Sprintf("%s/%s", source, packageName)
		if _, exists := graph.Nodes[depNodeID]; !exists {
			var depVersion models.Version
			if depVersion, err = version.Parse(depManifest.Version); err == nil {
				graph.Nodes[depNodeID] = &models.DependencyNode{
					Package: depPkg,
					Version: depVersion,
					ID:      depNodeID,
				}
			}
		}

		dep := models.Dependency{
			Source:  source,
			Package: packageName,
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

// Использует алгоритм DFS с отслеживанием рекурсивного стека.
// Сложность: O(V + E), где V - количество узлов, E - количество рёбер.
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
			err = ctx.Err()
			return
		default:
		}

		if !visited[nodeID] {
			if checkCycle(nodeID) {
				err = errors.New(i18n.Msg("Cycle detected in dependencies"))
				return
			}
		}
	}

	return
}

// SortForInstallation выполняет топологическую сортировку для установки.
func (r *resolver) SortForInstallation(ctx context.Context, graph *models.DependencyGraph) (packages []*models.Package, err error) {

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
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
			err = ctx.Err()
			return
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

	packages = result
	return
}

func (r *resolver) CheckCompatibility(ctx context.Context, installed *models.Package, required *models.Dependency) (compatible bool) {

	if required.Version == "" {
		return true
	}

	installation, err := r.databaseManager.FindByPackage(ctx, required.Source, installed.Name)
	if err != nil {
		return false
	}

	installedVersion, err := version.Parse(installation.Version)
	if err != nil {
		return false
	}

	return version.Match(required.Version, installedVersion)
}

// resolveSourceConflict разрешает конфликт источников для пакета интерактивно.
func (r *resolver) resolveSourceConflict(ctx context.Context, packageName string, requestedSource string) (pkg *models.Package, manifest *models.Manifest, err error) {

	// Получаем все варианты пакета из разных источников
	var allPackages []managers.PackageWithSource
	if allPackages, err = r.manifestManager.FindAllPackages(ctx, packageName); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get package information: %w"), err)
		return
	}

	if len(allPackages) == 0 {
		err = fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
		return
	}

	if len(allPackages) == 1 {
		// Если только один вариант, возвращаем его
		pkg = allPackages[0].Package
		manifest = allPackages[0].Manifest
		return
	}

	// Проверяем, есть ли уже установленный пакет
	var installed *models.Installation
	installed, err = r.databaseManager.FindByPackage(ctx, "", packageName)
	if err != nil {
		installed = nil
	}

	// Если указан запрошенный source, ищем пакет в этом источнике
	if requestedSource != "" {
		normalizedRequestedSource := storage.NormalizeSource(requestedSource)
		for _, pkgWithSource := range allPackages {
			normalizedCurrentSource := storage.NormalizeSource(pkgWithSource.Source)
			if normalizedRequestedSource == normalizedCurrentSource {
				// Найден пакет в запрошенном источнике
				// Проверяем версию, если есть установленный пакет
				if installed != nil {
					normalizedInstalledSource := storage.NormalizeSource(installed.Source)

					// Если источник совпадает с установленным, используем автоматически
					if normalizedInstalledSource == normalizedRequestedSource {
						pkg = pkgWithSource.Package
						manifest = pkgWithSource.Manifest
						return
					}

					// Если источник другой, проверяем версии
					var installedVersion models.Version
					if installedVersion, err = version.Parse(installed.Version); err == nil {
						var requestedVersion models.Version
						if requestedVersion, err = version.Parse(pkgWithSource.Manifest.Version); err == nil {
							comparison := version.Compare(installedVersion, requestedVersion)
							if comparison <= 0 {
								// Устанавливаемая версия >= установленной (upgrade или равная) - используем автоматически
								pkg = pkgWithSource.Package
								manifest = pkgWithSource.Manifest
								return
							}
							// Устанавливаемая версия < установленной (downgrade) - спрашиваем подтверждение
							confirmMessage := fmt.Sprintf(i18n.Msg("Replace package %s from source '%s' (version %s) with package from source '%s' (version %s)?"), packageName, installed.Source, installed.Version, pkgWithSource.Source, pkgWithSource.Manifest.Version)
							var confirm bool
							if confirm, err = pterm.DefaultInteractiveConfirm.
								WithDefaultValue(false).
								Show(confirmMessage); err != nil {
								err = fmt.Errorf(i18n.Msg("Confirmation cancelled: %w"), err)
								return
							}
							if !confirm {
								err = errors.New(i18n.Msg("Package replacement cancelled by user"))
								return
							}
							pkg = pkgWithSource.Package
							manifest = pkgWithSource.Manifest
							return
						}
					}
					// Если не удалось распарсить версию, спрашиваем подтверждение
					confirmMessage := fmt.Sprintf(i18n.Msg("Replace package %s from source '%s' (version %s) with package from source '%s' (version %s)?"), packageName, installed.Source, installed.Version, pkgWithSource.Source, pkgWithSource.Manifest.Version)
					var confirm bool
					if confirm, err = pterm.DefaultInteractiveConfirm.
						WithDefaultValue(false).
						Show(confirmMessage); err != nil {
						err = fmt.Errorf(i18n.Msg("Confirmation cancelled: %w"), err)
						return
					}
					if !confirm {
						err = errors.New(i18n.Msg("Package replacement cancelled by user"))
						return
					}
				}
				// Если нет установленного пакета, используем автоматически
				pkg = pkgWithSource.Package
				manifest = pkgWithSource.Manifest
				return
			}
		}
	}

	// Формируем информацию о конфликте
	var conflictMessage strings.Builder
	conflictMessage.WriteString(fmt.Sprintf(i18n.Msg("Package %s found in multiple sources:")+"\n", packageName))

	if installed != nil {
		conflictMessage.WriteString(fmt.Sprintf(i18n.Msg("Currently installed: %s (source: %s, version: %s)")+"\n", packageName, installed.Source, installed.Version))
	}

	conflictMessage.WriteString(i18n.Msg("Available sources:") + "\n")

	// Формируем опции для выбора
	options := make([]string, 0, len(allPackages))
	packageMap := make(map[string]*managers.PackageWithSource, len(allPackages))

	// Если указан запрошенный source, помечаем его
	if requestedSource != "" {
		conflictMessage.WriteString(fmt.Sprintf(i18n.Msg("Requested source: %s")+"\n", requestedSource))
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

		// Помечаем запрошенный source
		if requestedSource != "" && pkgWithSource.Source == requestedSource {
			optionText += " " + i18n.Msg("(requested)")
		}

		// Помечаем установленный пакет
		if installed != nil && installed.Source != "" {
			normalizedInstalledSource := strings.TrimSpace(installed.Source)
			normalizedCurrentSource := strings.TrimSpace(pkgWithSource.Source)
			if normalizedInstalledSource == normalizedCurrentSource {
				optionText += " " + i18n.Msg("(currently installed)")
			}
		}

		options = append(options, optionText)
		packageMap[optionText] = pkgWithSource
		conflictMessage.WriteString(fmt.Sprintf("  - %s\n", optionText))
	}

	// Выводим предупреждение о конфликте
	pterm.Warning.Println(conflictMessage.String())

	// Если есть установленный пакет из другого источника, предупреждаем о замене
	if installed != nil {
		// Проверяем, есть ли среди вариантов установленный пакет
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

	// Показываем интерактивный выбор
	var selected string
	if selected, err = pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(i18n.Msg("Select source for package")); err != nil || selected == "" {
		err = errors.New(i18n.Msg("Package source selection cancelled"))
		return
	}

	var selectedPkg *managers.PackageWithSource
	var exists bool
	if selectedPkg, exists = packageMap[selected]; !exists {
		err = errors.New(i18n.Msg("Selected package source not found"))
		return
	}

	// Если выбран другой источник, чем установленный, запрашиваем подтверждение
	if installed != nil && installed.Source != "" && selectedPkg.Source != installed.Source {
		confirmMessage := fmt.Sprintf(i18n.Msg("Replace package %s from source '%s' (version %s) with package from source '%s' (version %s)?"), packageName, installed.Source, installed.Version, selectedPkg.Source, selectedPkg.Manifest.Version)
		var confirm bool
		if confirm, err = pterm.DefaultInteractiveConfirm.
			WithDefaultValue(false).
			Show(confirmMessage); err != nil {
			err = fmt.Errorf(i18n.Msg("Confirmation cancelled: %w"), err)
			return
		}
		if !confirm {
			err = errors.New(i18n.Msg("Package replacement cancelled by user"))
			return
		}
	}

	pkg = selectedPkg.Package
	manifest = selectedPkg.Manifest
	return
}

// normalizeURI нормализует спецификацию пакета, преобразуя packageName@version в source:packageName@version.
// Если спецификация уже содержит URL, возвращает её как есть.
func (r *resolver) normalizeURI(ctx context.Context, spec string) (normalizedURI uri.URI, err error) {

	// Пытаемся распарсить как URI
	parsedURI, parseErr := uri.New(spec)
	if parseErr == nil {
		// Успешно распарсено - это уже валидный URI с URL
		normalizedURI = parsedURI
		return
	}

	// Если ошибка не "URL must have a scheme", возвращаем её
	if !strings.Contains(parseErr.Error(), "URL must have a scheme") {
		err = parseErr
		return
	}

	// Это packageName@version - нужно нормализовать
	// Извлекаем packageName и version
	parts := strings.Split(spec, "@")
	packageName := parts[0]
	versionStr := ""
	if len(parts) > 1 {
		versionStr = parts[1]
	}

	// Ищем пакет в загруженных манифестах
	var packages []managers.PackageWithSource
	if packages, err = r.manifestManager.FindAllPackages(ctx, packageName); err != nil {
		err = fmt.Errorf(i18n.Msg("Package %s not found: %w"), packageName, err)
		return
	}

	if len(packages) == 0 {
		err = fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
		return
	}

	// Если пакет найден в нескольких манифестах, берем первый
	// В будущем можно добавить интерактивный выбор
	source := packages[0].Source
	if source == "" {
		err = fmt.Errorf(i18n.Msg("Package %s found but source is empty"), packageName)
		return
	}

	// Формируем нормализованный URI: source:packageName@version
	normalizedSpec := source + ":" + packageName
	if versionStr != "" {
		normalizedSpec += "@" + versionStr
	}

	// Парсим нормализованный URI
	if normalizedURI, err = uri.New(normalizedSpec); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse normalized URI %s: %w"), normalizedSpec, err)
		return
	}

	return
}

// ensureManifestLoaded загружает манифест из source, если он еще не загружен.
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
		err = fmt.Errorf("failed to parse source URL: %w", err)
		return
	}

	var manifestURL string
	if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
		err = fmt.Errorf("failed to build manifest URL: %w", err)
		return
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
