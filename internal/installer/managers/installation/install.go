// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"github.com/tetratelabs/wazero"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
	"github.com/seniorGolang/tg/v3/internal/logger"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/ui"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/cache"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"
)

const (
	tempDirPrefix       = "draft"
	protocolFile        = "file://"
	packageVersionSep   = "@"
	extractedDirName    = "extracted"
	unknownVersion      = "unknown"
	noSourcePlaceholder = "(no source)"

	installStatusUnchanged = "unchanged"
	installStatusNew       = "new"
	installStatusUpdated   = "updated"
)

const (
	InstallStatusUnchanged = installStatusUnchanged
	InstallStatusNew       = installStatusNew
	InstallStatusUpdated   = installStatusUpdated
)

type SessionInstalledSet struct {
	IDs map[string]struct{}
}

type PackageDisplay struct {
	Name    string
	Display string
	Status  string
}

type TreeCollector interface {
	AddTree(source string, rootDisplay string, deps []PackageDisplay)
}

func (m *manager) Install(ctx context.Context, pkg *models.Package, v models.Version) (err error) {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	force := false
	if forceVal := ctx.Value(contextkeys.Force); forceVal != nil {
		if f, ok := forceVal.(bool); ok {
			force = f
		}
	}

	if !force {
		installID := m.generateInstallationID(pkg.Name, v.Original)
		var existingInstallation *models.Installation
		if existingInstallation, err = m.databaseManager.GetInstallation(ctx, installID); err == nil && existingInstallation != nil {
			var scopeErr error
			var scopeConfig *storage.ScopeConfig
			if scopeConfig, scopeErr = storage.LoadScopeConfig(m.scopeName); scopeErr == nil && m.verifyInstalledFilesChecksums(ctx, pkg, scopeConfig) {
				slog.Debug(i18n.Msg("Install: package already installed, skipping"), slog.String("package", pkg.Name), slog.String("version", v.Original), slog.String("id", installID))
				if collector := m.treeCollectorFromContext(ctx); collector != nil {
					m.printPackageProgressLine(pkg.Name)
					source := m.resolveTreeSourceEarly(ctx, pkg)
					rootDisplay := m.installStatusPrefix(installStatusUnchanged) + fmt.Sprintf("%s v%s", pkg.Name, v.Original)
					if pkg.Descr != "" {
						rootDisplay += " - " + pkg.Descr
					}
					collector.AddTree(source, rootDisplay, nil)
				}
				if skippedVal := ctx.Value(contextkeys.Skipped); skippedVal != nil {
					if skipped, ok := skippedVal.(*bool); ok {
						*skipped = true
					}
				}
				return
			}
		}
	}

	normalizedPkg := m.normalizePackage(ctx, pkg)

	var graph *models.DependencyGraph
	if graph, err = m.dependencyResolver.ResolveDependencies(ctx, normalizedPkg); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to resolve dependencies: %w"), err)
	}

	if err = m.dependencyResolver.CheckCycles(ctx, graph); err != nil {
		return fmt.Errorf(i18n.Msg("Circular dependencies detected: %w"), err)
	}

	var sortedPackages []*models.Package
	if sortedPackages, err = m.dependencyResolver.SortForInstallation(ctx, graph); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to sort packages for installation: %w"), err)
	}

	var allInstallations []models.Installation
	if allInstallations, err = m.databaseManager.ListInstallations(ctx); err != nil {
		allInstallations = []models.Installation{}
	}

	type packageToInstall struct {
		pkg    *models.Package
		v      models.Version
		source string
	}

	packagesToInstall := make([]packageToInstall, 0)
	packageStatus := make(map[string]string)

	var sessionSet *SessionInstalledSet
	if sessionVal := ctx.Value(contextkeys.SessionInstalledIDs); sessionVal != nil {
		sessionSet, _ = sessionVal.(*SessionInstalledSet)
	}

	findNodeByPackage := func(graph *models.DependencyGraph, pkg *models.Package) (node *models.DependencyNode) {

		for _, n := range graph.Nodes {
			if n != nil && n.Package == pkg {
				return n
			}
		}
		return nil
	}

	findVersionConstraint := func(pkgName string) (constraint string) {

		for _, edge := range graph.Edges {
			if edge.To != nil && edge.To.Package != nil && edge.To.Package.Name == pkgName {
				if edge.Dependency != nil && edge.Dependency.Version != "" {
					return edge.Dependency.Version
				}
			}
		}
		return ""
	}

	for _, depPkg := range sortedPackages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if depPkg.Name == pkg.Name {
			continue
		}

		var depSource string
		var depVersion models.Version
		if depNode := findNodeByPackage(graph, depPkg); depNode != nil {
			depVersion = depNode.Version
			depSource = depNode.Source
		}
		if depVersion.Original == "" {
			var depManifest *models.Manifest
			if _, depManifest, err = m.manifestManager.FindPackage(ctx, depPkg.Name); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to find manifest for dependency %s: %w"), depPkg.Name, err)
			}
			if depVersion, err = version.Parse(depManifest.Version); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to parse dependency version: %w"), err)
			}
		}

		versionConstraint := findVersionConstraint(depPkg.Name)

		var checkResult versionCheckResult
		if checkResult, err = m.checkPackageVersion(ctx, depPkg, depVersion, versionConstraint, allInstallations, depSource); err != nil {
			return
		}
		if !checkResult.shouldInstall {
			packageStatus[depPkg.Name] = installStatusUnchanged
			continue
		}

		depInstallID := m.generateInstallationID(depPkg.Name, depVersion.Original)
		if sessionSet != nil {
			if _, alreadyInstalled := sessionSet.IDs[depInstallID]; alreadyInstalled {
				packageStatus[depPkg.Name] = installStatusUnchanged
				continue
			}
		}

		packageStatus[depPkg.Name] = checkResult.installStatus
		packagesToInstall = append(packagesToInstall, packageToInstall{
			pkg:    depPkg,
			v:      depVersion,
			source: depSource,
		})
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var mainPkg *models.Package
	var rootNode *models.DependencyNode
	var rootSource string
	var mainVersion models.Version
	if pkg.Alias != "" {
		rootNode = graph.Nodes[pkg.Name]
	} else {
		rootNode = findNodeByPackage(graph, pkg)
	}
	if rootNode != nil {
		rootSource = rootNode.Source
		mainPkg = rootNode.Package
		mainVersion = rootNode.Version
	} else {
		mainPkg = pkg
		mainVersion = v
	}
	if mainVersion.Original == "" {
		mainVersion = v
	}
	var checkResultMain versionCheckResult
	if checkResultMain, err = m.checkPackageVersion(ctx, mainPkg, mainVersion, "", allInstallations, rootSource); err != nil {
		return
	}
	if !checkResultMain.shouldInstall {
		var scopeErr error
		var scopeConfig *storage.ScopeConfig
		if scopeConfig, scopeErr = storage.LoadScopeConfig(m.scopeName); scopeErr == nil && m.verifyInstalledFilesChecksums(ctx, mainPkg, scopeConfig) {
			if m.treeCollectorFromContext(ctx) != nil {
				m.printPackageProgressLine(mainPkg.Name)
			}
			if skippedVal := ctx.Value(contextkeys.Skipped); skippedVal != nil {
				if skipped, ok := skippedVal.(*bool); ok {
					*skipped = true
				}
			}
			treePackagesInfo := make([]packageToInstallInfo, 0)
			statusUnchanged := m.subtreePackageNames(graph, pkg.Name)
			for _, name := range statusUnchanged {
				packageStatus[name] = installStatusUnchanged
			}
			if collector := m.treeCollectorFromContext(ctx); collector != nil {
				source := m.resolveTreeSource(ctx, graph, pkg.Name)
				rootDisplay, deps := m.buildPackageFlatInfo(graph, packageStatus, pkg.Name, v)
				collector.AddTree(source, rootDisplay, deps)
			} else {
				m.printDependencyTree(ctx, graph, treePackagesInfo, packageStatus, pkg.Name, v)
			}
			return
		}
	}

	rootInstallID := m.generateInstallationID(mainPkg.Name, mainVersion.Original)
	if sessionSet != nil {
		if _, alreadyInstalled := sessionSet.IDs[rootInstallID]; alreadyInstalled {
			packageStatus[mainPkg.Name] = installStatusUnchanged
		} else {
			packagesToInstall = append(packagesToInstall, packageToInstall{
				pkg:    mainPkg,
				v:      mainVersion,
				source: rootSource,
			})
			packageStatus[mainPkg.Name] = checkResultMain.installStatus
		}
	} else {
		packagesToInstall = append(packagesToInstall, packageToInstall{
			pkg:    mainPkg,
			v:      mainVersion,
			source: rootSource,
		})
		packageStatus[mainPkg.Name] = checkResultMain.installStatus
	}

	packagesInfo := make([]packageToInstallInfo, 0, len(packagesToInstall))
	for _, pkgToInstall := range packagesToInstall {
		packagesInfo = append(packagesInfo, packageToInstallInfo{
			pkgName: pkgToInstall.pkg.Name,
			v:       pkgToInstall.v,
		})
	}
	if collector := m.treeCollectorFromContext(ctx); collector != nil {
		source := m.resolveTreeSource(ctx, graph, pkg.Name)
		rootDisplay, deps := m.buildPackageFlatInfo(graph, packageStatus, pkg.Name, v)
		collector.AddTree(source, rootDisplay, deps)
	} else {
		m.printDependencyTree(ctx, graph, packagesInfo, packageStatus, pkg.Name, v)
	}

	for _, pkgToInstall := range packagesToInstall {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err = m.installPackage(ctx, pkgToInstall.pkg, pkgToInstall.v, pkgToInstall.source); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to install package %s: %w"), pkgToInstall.pkg.Name, err)
		}

		if sessionSet != nil {
			installID := m.generateInstallationID(pkgToInstall.pkg.Name, pkgToInstall.v.Original)
			sessionSet.IDs[installID] = struct{}{}
		}
	}

	return
}

func (m *manager) installPackage(ctx context.Context, pkg *models.Package, v models.Version, downloadSource string) (err error) {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var source string
	if downloadSource != "" {
		source = downloadSource
	}
	// Подмена источника контекстом только для пакетов с alias: подставляем источник манифеста.
	if pkg.Alias != "" {
		if ctxSource := ctx.Value(contextkeys.Source); ctxSource != nil {
			if s, ok := ctxSource.(string); ok && s != "" {
				source = s
			}
		}
	}
	if source == "" {
		if source, err = m.findPackageSource(ctx, pkg); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to find package source: %w"), err)
		}
	}

	if err = m.checkFileConflicts(ctx, pkg, source, v.Original); err != nil {
		return
	}

	var scopeConfig *storage.ScopeConfig
	if scopeConfig, err = storage.LoadScopeConfig(m.scopeName); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to load scope configuration: %w"), err)
	}

	pterm.Debug.Printf(i18n.Msg("Scope config - InstallPrefix: %s, BinDir: %s")+"\n", scopeConfig.InstallPrefix, scopeConfig.BinDir)

	normalizedSource := storage.BaseSourceURL(source)

	installID := m.generateInstallationID(pkg.Name, v.Original)
	tempDir := filepath.Join(os.TempDir(), tempDirPrefix, installID)
	defer os.RemoveAll(tempDir)

	if err = os.MkdirAll(tempDir, defaultDirMode); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "temporary directory", err)
	}

	if pkg.Scripts != nil && pkg.Scripts.PreInstall != nil {
		if err = m.executeScript(ctx, pkg.Scripts.PreInstall, tempDir, nil); err != nil {
			return fmt.Errorf(i18n.Msg("Error executing pre_install script: %w"), err)
		}
	}

	extractedDirs := make(map[string]string)
	downloadedFiles := make(map[string]string)

	var packageBar *ui.ProgressBar
	var hasProgressWork bool
	defer func() {

		if hasProgressWork && packageBar != nil {
			fmt.Println("\r" + packageBar.Stop())
		}
	}()

	ensureProgressBar := func() {

		if packageBar == nil {
			packageTitle := fmt.Sprintf(i18n.Msg("Installing %s"), pkg.Name)
			maxTitleLength := len(packageTitle)
			packageBar = ui.NewProgressBar(packageTitle, 100, maxTitleLength)
		}
		hasProgressWork = true
		packageBar.SetCurrent(0)
		packageBar.Print()
	}

	if m.treeCollectorFromContext(ctx) != nil {
		ensureProgressBar()
	}

	for _, download := range pkg.Downloads {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var url string
		if url, err = m.selectURLForDownload([]models.PlatformDownload{download}); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to select URL for download: %w"), err)
		}

		fileName := filepath.Base(url)
		destPath := filepath.Join(tempDir, fileName)

		if m.isArchive(destPath) {
			ensureProgressBar()
			if err = m.downloadWithProgress(ctx, url, destPath, packageBar); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to download file %s: %w"), url, err)
			}
			downloadedFiles[fileName] = destPath
			extractDir := filepath.Join(tempDir, extractedDirName, fileName)
			if err = m.extractArchive(ctx, destPath, extractDir); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to extract archive %s: %w"), fileName, err)
			}
			extractedDirs[fileName] = extractDir
			continue
		}

		fileInst := m.findFileInstForDownload(pkg, fileName)
		if fileInst != nil && fileInst.Checksum != "" {
			destination := m.resolveDestination(fileInst.Destination, scopeConfig)
			if _, statErr := os.Stat(destination); statErr == nil {
				parts := strings.Split(fileInst.Checksum, ":")
				if len(parts) == 2 {
					if m.validationEngine.ValidateChecksum(ctx, destination, parts[0], parts[1]) == nil {
						downloadedFiles[fileName] = destination
						continue
					}
				}
			}
		}

		ensureProgressBar()
		if err = m.downloadWithProgress(ctx, url, destPath, packageBar); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to download file %s: %w"), url, err)
		}
		downloadedFiles[fileName] = destPath
	}

	installedFiles := make([]models.InstalledFile, 0)

	for _, fileInst := range pkg.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fileName := fileInst.File
		if fileName == "" && len(pkg.Downloads) > 0 {
			var url string
			if url, err = m.selectURLForDownload(pkg.Downloads); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to select URL to determine file name: %w"), err)
			}
			fileName = filepath.Base(url)
		}

		var sourcePath string
		if sourcePath, err = m.findSourceFile(fileName, fileInst.Source, downloadedFiles, extractedDirs); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to find source file: %w"), err)
		}

		destination := m.resolveDestination(fileInst.Destination, scopeConfig)

		if err = os.MkdirAll(filepath.Dir(destination), defaultDirMode); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
		}

		var actualSourcePath string
		if fileInst.Source != "" {
			potentialPath := filepath.Join(filepath.Dir(sourcePath), fileInst.Source)
			var statErr error
			if _, statErr = os.Stat(potentialPath); statErr == nil {
				actualSourcePath = potentialPath
			} else {
				actualSourcePath = sourcePath
			}
		} else {
			actualSourcePath = sourcePath
		}

		skipCopy := false
		if actualSourcePath == destination {
			skipCopy = true
		} else {
			absSource, absDest := actualSourcePath, destination
			if a, e := filepath.Abs(actualSourcePath); e == nil {
				absSource = a
			}
			if a, e := filepath.Abs(destination); e == nil {
				absDest = a
			}
			if absSource == absDest {
				skipCopy = true
			}
		}

		if !skipCopy {
			ensureProgressBar()
			if err = m.copyFileWithProgress(actualSourcePath, destination, fileInst.Source, packageBar); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
			}
		}

		var info os.FileInfo
		if info, err = os.Stat(destination); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to get file information: %w"), err)
		}

		if fileInst.Checksum != "" && !skipCopy {
			parts := strings.Split(fileInst.Checksum, ":")
			if len(parts) == 2 {
				if err = m.validationEngine.ValidateChecksum(ctx, destination, parts[0], parts[1]); err != nil {
					return fmt.Errorf(i18n.Msg("Checksum validation failed: %w"), err)
				}
			}
		}

		installedFiles = append(installedFiles, models.InstalledFile{
			Path:     destination,
			Source:   fileInst.Source,
			Checksum: fileInst.Checksum,
			Size:     info.Size(),
		})
	}

	if pkg.Scripts != nil && pkg.Scripts.PostInstall != nil {
		if err = m.executeScript(ctx, pkg.Scripts.PostInstall, tempDir, extractedDirs); err != nil {
			return fmt.Errorf(i18n.Msg("Error executing post_install script: %w"), err)
		}
	}

	installation := &models.Installation{
		ID:           installID,
		Source:       normalizedSource,
		Package:      pkg.Name,
		Version:      v.Original,
		Descr:        pkg.Descr,
		InstalledAt:  time.Now(),
		Files:        installedFiles,
		Dependencies: pkg.Dependencies,
	}

	var wasmFile *models.InstalledFile
	for _, file := range installedFiles {
		if strings.HasSuffix(file.Path, plugin.FileExtTGP) {
			wasmFile = &file

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var workDone chan struct{}
			if packageBar != nil {
				workDone = make(chan struct{})
				packageBar.SetIndeterminate(true)
				go func() {
					ticker := time.NewTicker(80 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-workDone:
							return
						case <-ticker.C:
							packageBar.Print()
						}
					}
				}()
				defer func() { close(workDone) }()
			}

			var readErr error
			var rawBytes []byte
			if rawBytes, readErr = os.ReadFile(file.Path); readErr != nil {
				return fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), readErr)
			}

			var wasmBytes []byte
			if wasmBytes, err = plugin.DecodeTGPBytes(rawBytes); err != nil {
				return
			}

			loggerAdapter := logger.NewSlogAdapter(slog.Default())

			var cacheErr error
			var compilationCache wazero.CompilationCache
			compilationCache, cacheErr = cache.GetCompilationCache(ctx)
			if cacheErr != nil {
				slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), slog.Any("error", cacheErr))
			}

			var hostErr error
			var tempHost *host.Host
			tgPath := scopeConfig.ConfigDir
			if compilationCache != nil {
				tempHost, hostErr = wasm.New(ctx, wasmBytes, plugin.Info{}, ".", loggerAdapter, wasm.WithCompilationCache(compilationCache), wasm.WithTGPath(tgPath), wasm.MuteLogs())
			} else {
				tempHost, hostErr = wasm.New(ctx, wasmBytes, plugin.Info{}, ".", loggerAdapter, wasm.WithTGPath(tgPath), wasm.MuteLogs())
			}
			if hostErr != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", hostErr)
			}
			defer wasm.Close(ctx, tempHost)

			var info plugin.Info
			var infoErr error
			if info, infoErr = imports.Info(ctx, tempHost); infoErr != nil {
				return fmt.Errorf(i18n.Msg("failed to get plugin info: %w"), infoErr)
			}

			if info.Name == "" {
				return errors.New(i18n.Msg("invalid plugin: missing name"))
			}

			if info.Name != pkg.Name {
				return fmt.Errorf("%s: %s != %s", i18n.Msg("plugin name does not match package name"), info.Name, pkg.Name)
			}

			installation.Commands = convertCommandsToModel(info.Commands)
			installation.Options = convertOptionsToModel(info.Options)
			installation.Kind = info.Kind
			installation.Silent = info.Silent
			installation.Always = info.Always
			installation.InitPkgs = info.InitPkgs
			installation.AllowedHosts = info.AllowedHosts
			installation.AllowedListeners = info.AllowedListeners
			installation.AllowedPaths = info.AllowedPaths
			installation.Dependencies = info.Dependencies
			installation.AllowedEnvVars = info.AllowedEnvVars
			installation.AllowedShellCMDs = info.AllowedShellCMDs
			installation.AllowedStdOut = info.AllowedStdOut
			installation.AllowedStdErr = info.AllowedStdErr

			break
		}
	}

	if len(installedFiles) > 0 && wasmFile == nil {
		return errors.New(i18n.Msg("package appears to be a plugin but no .tgp file found"))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err = m.databaseManager.RecordInstallation(ctx, installation); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to record installation in database: %w"), err)
	}

	return
}

// generateInstallationID: детерминированный UUIDv5 по Package+Version. Один пакет+версия — один ID; источник не участвует — переустановка из другого источника заменяет запись.
func (m *manager) generateInstallationID(packageName string, version string) (id string) {

	name := packageName + packageVersionSep + version
	idUUID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(name))

	return idUUID.String()
}

func (m *manager) findFileInstForDownload(pkg *models.Package, fileName string) (fileInst *models.FileInstallation) {

	for i := range pkg.Files {
		fi := &pkg.Files[i]
		if fi.File == fileName {
			return fi
		}
	}
	if len(pkg.Files) == 1 && len(pkg.Downloads) == 1 && pkg.Files[0].File == "" {
		return &pkg.Files[0]
	}
	return nil
}

func (m *manager) findSourceFile(fileName string, sourcePath string, downloadedFiles map[string]string, extractedDirs map[string]string) (path string, err error) {

	var exists bool
	var extractedDir string
	if extractedDir, exists = extractedDirs[fileName]; exists {
		if sourcePath != "" {
			fullPath := filepath.Join(extractedDir, sourcePath)
			if _, statErr := os.Stat(fullPath); statErr == nil {
				path = fullPath
				return
			}
		}

		var downloadedFile string
		if downloadedFile, exists = downloadedFiles[fileName]; exists {
			baseName := filepath.Base(downloadedFile)
			fullPath := filepath.Join(extractedDir, baseName)
			if _, statErr := os.Stat(fullPath); statErr == nil {
				path = fullPath
				return
			}
		}
	}

	var downloadedFile string
	if downloadedFile, exists = downloadedFiles[fileName]; exists {
		if sourcePath != "" {
			path = filepath.Join(filepath.Dir(downloadedFile), sourcePath)
			return
		}
		path = downloadedFile
		return
	}

	return "", fmt.Errorf(i18n.Msg("File %s not found"), fileName)
}

func (m *manager) subtreePackageNames(graph *models.DependencyGraph, rootPackageName string) (names []string) {

	var rootNode *models.DependencyNode
	for _, n := range graph.Nodes {
		if n.Package.Name == rootPackageName {
			rootNode = n
			break
		}
	}
	if rootNode == nil {
		return nil
	}

	visited := make(map[string]bool)
	queue := []*models.DependencyNode{rootNode}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if visited[node.ID] {
			continue
		}
		visited[node.ID] = true
		names = append(names, node.Package.Name)
		for _, edge := range graph.Edges {
			if edge.From.ID == node.ID && edge.To != nil && !visited[edge.To.ID] {
				queue = append(queue, edge.To)
			}
		}
	}
	return
}

func (m *manager) resolveTreeSourceEarly(ctx context.Context, pkg *models.Package) (source string) {

	if sourceVal := ctx.Value(contextkeys.Source); sourceVal != nil {
		if s, ok := sourceVal.(string); ok && s != "" {
			return s
		}
	}
	var err error
	if source, err = m.findPackageSource(ctx, pkg); err != nil {
		return noSourcePlaceholder
	}
	return
}

func (m *manager) printPackageProgressLine(packageName string) {

	packageTitle := fmt.Sprintf(i18n.Msg("Installing %s"), packageName)
	maxTitleLength := len(packageTitle)
	bar := ui.NewProgressBar(packageTitle, 100, maxTitleLength)
	bar.SetCurrent(100)
	fmt.Println("\r" + bar.Stop())
}

func (m *manager) resolveDestination(destination string, scopeConfig *storage.ScopeConfig) (path string) {

	destination = strings.ReplaceAll(destination, "${OS}", runtime.GOOS)
	destination = strings.ReplaceAll(destination, "${ARCH}", runtime.GOARCH)

	return filepath.Join(scopeConfig.InstallPrefix, destination)
}

// verifyInstalledFilesChecksums: проверка файлов по контрольным суммам; ошибки только в debug; false при отсутствии файла или несовпадении суммы.
func (m *manager) verifyInstalledFilesChecksums(ctx context.Context, pkg *models.Package, scopeConfig *storage.ScopeConfig) (allOK bool) {

	if scopeConfig == nil {
		slog.Debug(i18n.Msg("verifyInstalledFilesChecksums: scope config is nil"))
		return false
	}

	for _, fileInst := range pkg.Files {
		if fileInst.Checksum == "" {
			continue
		}

		destination := m.resolveDestination(fileInst.Destination, scopeConfig)
		if _, statErr := os.Stat(destination); statErr != nil {
			slog.Debug(i18n.Msg("Checksum verification failed: file missing or unreadable"),
				slog.String("package", pkg.Name),
				slog.String("path", destination),
				slog.Any("error", statErr))
			return false
		}

		parts := strings.Split(fileInst.Checksum, ":")
		if len(parts) != 2 {
			slog.Debug(i18n.Msg("Checksum verification failed: invalid checksum format"),
				slog.String("package", pkg.Name),
				slog.String("path", destination))
			return false
		}

		if m.validationEngine.ValidateChecksum(ctx, destination, parts[0], parts[1]) != nil {
			slog.Debug(i18n.Msg("Checksum verification failed: checksum mismatch"),
				slog.String("package", pkg.Name),
				slog.String("path", destination))
			return false
		}
	}

	return true
}

// normalizePackage: заполняет пустые source в зависимостях из source основного пакета.
func (m *manager) normalizePackage(ctx context.Context, pkg *models.Package) (normalized *models.Package) {

	mainSource := ""
	if sourceVal := ctx.Value(contextkeys.Source); sourceVal != nil {
		if source, ok := sourceVal.(string); ok {
			mainSource = source
		}
	}

	if mainSource == "" {
		var err error
		if mainSource, err = m.findPackageSource(ctx, pkg); err != nil {
			return pkg
		}
	}

	needsNormalization := false
	for _, depStr := range pkg.Dependencies {
		var parseErr error
		var parsedURI uri.URI
		if parsedURI, parseErr = uri.New(depStr); parseErr != nil {
			continue
		}
		if parsedURI.Source() == "" {
			needsNormalization = true
			break
		}
	}

	if !needsNormalization {
		return pkg
	}

	normalized = &models.Package{
		Name:         pkg.Name,
		Descr:        pkg.Descr,
		Alias:        pkg.Alias,
		Files:        pkg.Files,
		Downloads:    pkg.Downloads,
		Dependencies: make([]string, len(pkg.Dependencies)),
		Scripts:      pkg.Scripts,
	}

	for i, depStr := range pkg.Dependencies {
		var parseErr error
		var parsedURI uri.URI
		if parsedURI, parseErr = uri.New(depStr); parseErr != nil {
			normalized.Dependencies[i] = depStr
			continue
		}

		source := parsedURI.Source()
		packageName := parsedURI.Package()
		ver := parsedURI.Version().Original

		if source == "" {
			if ver != "" {
				normalized.Dependencies[i] = mainSource + ":" + packageName + "@" + ver
			} else {
				normalized.Dependencies[i] = mainSource + ":" + packageName
			}
		} else {
			normalized.Dependencies[i] = depStr
		}
	}

	return
}

func (m *manager) downloadWithProgress(ctx context.Context, url string, destination string, bar *ui.ProgressBar) (err error) {

	if strings.HasPrefix(url, protocolFile) {
		return m.downloadManager.Download(ctx, url, destination)
	}

	progressChan := make(chan int, 10)
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.downloadManager.DownloadWithProgress(ctx, url, destination, progressChan)
	}()

	var downloadErr error

	for {
		select {
		case percent, ok := <-progressChan:
			if !ok {
				bar.SetCurrent(100)
				bar.Print()
				if downloadErr != nil {
					return downloadErr
				}
				select {
				case downloadErr = <-errChan:
					return downloadErr
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			bar.SetCurrent(percent)
			bar.Print()
		case downloadErr = <-errChan:
			if downloadErr != nil {
				return downloadErr
			}
			select {
			case _, ok := <-progressChan:
				_ = ok // Игнорируем значение, просто проверяем закрытие канала
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			bar.SetCurrent(100)
			bar.Print()
			return
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type packageToInstallInfo struct {
	pkgName string
	v       models.Version
}

func (m *manager) treeCollectorFromContext(ctx context.Context) (c TreeCollector) {

	if v := ctx.Value(contextkeys.TreeCollector); v != nil {
		c, _ = v.(TreeCollector)
	}
	return
}

func (m *manager) resolveTreeSource(ctx context.Context, graph *models.DependencyGraph, rootPackageName string) (source string) {

	if sourceVal := ctx.Value(contextkeys.Source); sourceVal != nil {
		if s, ok := sourceVal.(string); ok && s != "" {
			return s
		}
	}
	var rootPkg *models.Package
	for _, node := range graph.Nodes {
		if node.Package.Name == rootPackageName {
			rootPkg = node.Package
			break
		}
	}
	if rootPkg == nil {
		return noSourcePlaceholder
	}
	var err error
	if source, err = m.findPackageSource(ctx, rootPkg); err != nil {
		return noSourcePlaceholder
	}
	return
}

func (m *manager) buildPackageFlatInfo(graph *models.DependencyGraph, packageStatus map[string]string, rootPackageName string, rootVersion models.Version) (rootDisplay string, deps []PackageDisplay) {

	var rootNode *models.DependencyNode
	for _, n := range graph.Nodes {
		if n.Package.Name == rootPackageName {
			rootNode = n
			break
		}
	}
	if rootNode == nil {
		return "", nil
	}

	status := packageStatus[rootNode.Package.Name]
	if status == "" {
		status = installStatusUnchanged
	}
	rootDisplay = m.installStatusPrefix(status) + fmt.Sprintf("%s v%s", rootNode.Package.Name, getVersionString(rootNode.Version, rootVersion))
	if rootNode.Package.Descr != "" {
		rootDisplay += " - " + rootNode.Package.Descr
	}

	seen := make(map[string]bool)
	queue := []*models.DependencyNode{rootNode}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, edge := range graph.Edges {
			if edge.From.ID != node.ID || edge.To == nil {
				continue
			}
			child := edge.To
			if seen[child.ID] {
				continue
			}
			seen[child.ID] = true
			st := packageStatus[child.Package.Name]
			if st == "" {
				st = installStatusUnchanged
			}
			display := m.installStatusPrefix(st) + fmt.Sprintf("%s v%s", child.Package.Name, getVersionString(child.Version, rootVersion))
			if child.Package.Descr != "" {
				display += " - " + child.Package.Descr
			}
			deps = append(deps, PackageDisplay{Name: child.Package.Name, Display: display, Status: st})
			queue = append(queue, child)
		}
	}
	return
}

func (m *manager) installStatusPrefix(status string) (prefix string) {

	switch status {
	case installStatusUnchanged:
		return pterm.Cyan("✓") + " "
	case installStatusUpdated:
		return pterm.Yellow("↑") + " "
	case installStatusNew:
		return pterm.Green("●") + " "
	default:
		return pterm.Cyan("✓") + " "
	}
}

func (m *manager) buildPackageSubtree(graph *models.DependencyGraph, packagesInfo []packageToInstallInfo, packageStatus map[string]string, rootPackageName string, rootVersion models.Version) (node pterm.TreeNode) {

	packagesToInstallMap := make(map[string]models.Version)
	for _, pkgInfo := range packagesInfo {
		packagesToInstallMap[pkgInfo.pkgName] = pkgInfo.v
	}

	var buildPackageTreeNode func(n *models.DependencyNode, visited map[string]bool) pterm.TreeNode
	buildPackageTreeNode = func(n *models.DependencyNode, visited map[string]bool) (treeNode pterm.TreeNode) {

		if visited[n.ID] {
			status := packageStatus[n.Package.Name]
			if status == "" {
				status = installStatusUnchanged
			}
			visitedText := m.installStatusPrefix(status) + fmt.Sprintf("%s v%s", n.Package.Name, getVersionString(n.Version, rootVersion))
			if n.Package.Descr != "" {
				visitedText += " - " + n.Package.Descr
			}
			return pterm.TreeNode{Text: visitedText}
		}
		visited[n.ID] = true

		status := packageStatus[n.Package.Name]
		if status == "" {
			status = installStatusUnchanged
		}
		nodeText := m.installStatusPrefix(status) + fmt.Sprintf("%s v%s", n.Package.Name, getVersionString(n.Version, rootVersion))
		if n.Package.Descr != "" {
			nodeText += " - " + n.Package.Descr
		}

		var children []pterm.TreeNode
		for _, edge := range graph.Edges {
			if edge.From.ID == n.ID && edge.To != nil {
				children = append(children, buildPackageTreeNode(edge.To, visited))
			}
		}
		return pterm.TreeNode{Text: nodeText, Children: children}
	}

	var rootNode *models.DependencyNode
	for _, n := range graph.Nodes {
		if n.Package.Name == rootPackageName {
			rootNode = n
			break
		}
	}
	if rootNode == nil {
		return pterm.TreeNode{}
	}
	return buildPackageTreeNode(rootNode, make(map[string]bool))
}

func (m *manager) printDependencyTree(ctx context.Context, graph *models.DependencyGraph, packagesInfo []packageToInstallInfo, packageStatus map[string]string, rootPackageName string, rootVersion models.Version) {

	source := m.resolveTreeSource(ctx, graph, rootPackageName)
	subtree := m.buildPackageSubtree(graph, packagesInfo, packageStatus, rootPackageName, rootVersion)
	rootTreeNode := pterm.TreeNode{
		Text:     source,
		Children: []pterm.TreeNode{subtree},
	}
	pterm.Println()
	if err := pterm.DefaultTree.WithRoot(rootTreeNode).Render(); err != nil {
		slog.Debug(i18n.Msg("Failed to render dependency tree"), slog.Any("error", err))
	}
}

func getVersionString(v models.Version, defaultVersion models.Version) (versionStr string) {

	if v.Original != "" {
		return v.Original
	}
	if defaultVersion.Original != "" {
		return defaultVersion.Original
	}
	return unknownVersion
}
