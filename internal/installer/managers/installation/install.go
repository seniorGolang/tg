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
	tempDirPrefix     = "draft"
	protocolFile      = "file://"
	packageVersionSep = "@"
	extractedDirName  = "extracted"
	unknownVersion    = "unknown"
	// noSourcePlaceholder плейсхолдер для отсутствующего источника
	noSourcePlaceholder = "(no source)"
)

func (m *manager) Install(ctx context.Context, pkg *models.Package, v models.Version) (err error) {

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	force := false
	if forceVal := ctx.Value(contextkeys.Force); forceVal != nil {
		if f, ok := forceVal.(bool); ok {
			force = f
		}
	}

	// Если не указан --force, проверяем, существует ли установка с таким ID
	// ID формируется из package@version, поэтому один и тот же пакет с одной версией всегда имеет один ID
	if !force {
		installID := m.generateInstallationID(pkg.Name, v.Original)
		var existingInstallation *models.Installation
		if existingInstallation, err = m.databaseManager.GetInstallation(ctx, installID); err == nil && existingInstallation != nil {
			// Установка с таким ID уже существует - пропускаем установку
			slog.Debug(i18n.Msg("Install: package already installed, skipping"), slog.String("package", pkg.Name), slog.String("version", v.Original), slog.String("id", installID))
			// Помечаем в контексте, что пакет был пропущен
			if skippedVal := ctx.Value(contextkeys.Skipped); skippedVal != nil {
				if skipped, ok := skippedVal.(*bool); ok {
					*skipped = true
				}
			}
			return
		}
	}

	// Нормализуем пакет: заполняем пустые source в зависимостях source основного пакета
	normalizedPkg := m.normalizePackage(ctx, pkg)

	var graph *models.DependencyGraph
	if graph, err = m.dependencyResolver.ResolveDependencies(ctx, normalizedPkg); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to resolve dependencies: %w"), err)
		return
	}

	if err = m.dependencyResolver.CheckCycles(ctx, graph); err != nil {
		err = fmt.Errorf(i18n.Msg("Circular dependencies detected: %w"), err)
		return
	}

	var sortedPackages []*models.Package
	if sortedPackages, err = m.dependencyResolver.SortForInstallation(ctx, graph); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to sort packages for installation: %w"), err)
		return
	}

	// Загружаем актуальный список установок после разрешения зависимостей,
	// чтобы учитывать зависимости, установленные при установке предыдущих пакетов
	var allInstallations []models.Installation
	if allInstallations, err = m.databaseManager.ListInstallations(ctx); err != nil {
		allInstallations = []models.Installation{}
	}

	// Структура для хранения информации о пакетах для установки
	type packageToInstall struct {
		pkg    *models.Package
		v      models.Version
		source string
	}

	packagesToInstall := make([]packageToInstall, 0)

	findNodeByPackage := func(graph *models.DependencyGraph, pkg *models.Package) (node *models.DependencyNode) {
		for _, n := range graph.Nodes {
			if n != nil && n.Package == pkg {
				return n
			}
		}
		return nil
	}

	// findVersionConstraint находит требование версии для пакета из графа зависимостей.
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

	// Сначала собираем все подтверждения для зависимостей
	for _, depPkg := range sortedPackages {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
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
				err = fmt.Errorf(i18n.Msg("Failed to find manifest for dependency %s: %w"), depPkg.Name, err)
				return
			}
			if depVersion, err = version.Parse(depManifest.Version); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to parse dependency version: %w"), err)
				return
			}
		}

		versionConstraint := findVersionConstraint(depPkg.Name)

		// При проверке зависимостей создаем контекст БЕЗ флага --force,
		// чтобы зависимости проверялись по установленным версиям и не переустанавливались,
		// если уже установлены с подходящей версией. Флаг --force применяется только к явно указанным пакетам.
		depCtx := context.WithValue(ctx, contextkeys.Force, false)

		var checkResult versionCheckResult
		if checkResult, err = m.checkPackageVersion(depCtx, depPkg, depVersion, versionConstraint, allInstallations, depSource); err != nil {
			return
		}
		if !checkResult.shouldInstall {
			// Пакет уже установлен с подходящей версией - пропускаем
			continue
		}

		packagesToInstall = append(packagesToInstall, packageToInstall{
			pkg:    depPkg,
			v:      depVersion,
			source: depSource,
		})
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	var rootNode *models.DependencyNode
	if pkg.Alias != "" {
		rootNode = graph.Nodes[pkg.Name]
	} else {
		rootNode = findNodeByPackage(graph, pkg)
	}
	var rootSource string
	var mainPkg *models.Package
	var mainVersion models.Version
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
		// Пакет уже установлен с такой же версией - пропускаем
		// Помечаем в контексте, что пакет был пропущен
		if skippedVal := ctx.Value(contextkeys.Skipped); skippedVal != nil {
			if skipped, ok := skippedVal.(*bool); ok {
				*skipped = true
			}
		}
		// Выводим дерево зависимостей даже если пакет уже установлен
		packagesInfo := make([]packageToInstallInfo, 0)
		m.printDependencyTree(ctx, graph, packagesInfo, pkg.Name, v)
		return
	}

	packagesToInstall = append(packagesToInstall, packageToInstall{
		pkg:    mainPkg,
		v:      mainVersion,
		source: rootSource,
	})

	// Выводим дерево зависимостей перед установкой (всегда)
	packagesInfo := make([]packageToInstallInfo, 0, len(packagesToInstall))
	for _, pkgToInstall := range packagesToInstall {
		packagesInfo = append(packagesInfo, packageToInstallInfo{
			pkgName: pkgToInstall.pkg.Name,
			v:       pkgToInstall.v,
		})
	}
	m.printDependencyTree(ctx, graph, packagesInfo, pkg.Name, v)

	// Теперь устанавливаем все пакеты из списка
	for _, pkgToInstall := range packagesToInstall {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err = m.installPackage(ctx, pkgToInstall.pkg, pkgToInstall.v, pkgToInstall.source); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to install package %s: %w"), pkgToInstall.pkg.Name, err)
			return
		}
	}

	return
}

func (m *manager) installPackage(ctx context.Context, pkg *models.Package, v models.Version, downloadSource string) (err error) {

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	var source string
	if downloadSource != "" {
		source = downloadSource
	}
	if ctxSource := ctx.Value(contextkeys.Source); ctxSource != nil {
		if s, ok := ctxSource.(string); ok && s != "" {
			source = s
		}
	}
	if source == "" {
		if source, err = m.findPackageSource(ctx, pkg); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to find package source: %w"), err)
			return
		}
	}

	if err = m.checkFileConflicts(ctx, pkg, source, v.Original); err != nil {
		return
	}

	var scopeConfig *storage.ScopeConfig
	if scopeConfig, err = storage.LoadScopeConfig(m.scopeName); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to load scope configuration: %w"), err)
		return
	}

	pterm.Debug.Printf(i18n.Msg("Scope config - InstallPrefix: %s, BinDir: %s")+"\n", scopeConfig.InstallPrefix, scopeConfig.BinDir)

	normalizedSource := storage.NormalizeSourceForInstallation(source)

	// Генерируем детерминированный ID на основе Package + Version (UUIDv5).
	installID := m.generateInstallationID(pkg.Name, v.Original)
	tempDir := filepath.Join(os.TempDir(), tempDirPrefix, installID)
	defer os.RemoveAll(tempDir)

	if err = os.MkdirAll(tempDir, defaultDirMode); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "temporary directory", err)
		return
	}

	if pkg.Scripts != nil && pkg.Scripts.PreInstall != nil {
		if err = m.executeScript(ctx, pkg.Scripts.PreInstall, tempDir, nil); err != nil {
			err = fmt.Errorf(i18n.Msg("Error executing pre_install script: %w"), err)
			return
		}
	}

	extractedDirs := make(map[string]string)
	downloadedFiles := make(map[string]string)

	packageTitle := fmt.Sprintf(i18n.Msg("Installing %s"), pkg.Name)
	maxTitleLength := len(packageTitle)
	packageBar := ui.NewProgressBar(packageTitle, 100, maxTitleLength)
	defer func() {
		fmt.Println("\r" + packageBar.Stop())
	}()

	for _, download := range pkg.Downloads {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var url string
		if url, err = m.selectURLForDownload([]models.PlatformDownload{download}); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to select URL for download: %w"), err)
			return
		}

		fileName := filepath.Base(url)
		destPath := filepath.Join(tempDir, fileName)

		if m.isArchive(destPath) {
			packageBar.SetCurrent(0)
			packageBar.Print()
			if err = m.downloadWithProgress(ctx, url, destPath, packageBar); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to download file %s: %w"), url, err)
				return
			}
			downloadedFiles[fileName] = destPath
			extractDir := filepath.Join(tempDir, extractedDirName, fileName)
			if err = m.extractArchive(ctx, destPath, extractDir); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to extract archive %s: %w"), fileName, err)
				return
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

		packageBar.SetCurrent(0)
		packageBar.Print()
		if err = m.downloadWithProgress(ctx, url, destPath, packageBar); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to download file %s: %w"), url, err)
			return
		}
		downloadedFiles[fileName] = destPath
	}

	installedFiles := make([]models.InstalledFile, 0)

	for _, fileInst := range pkg.Files {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		fileName := fileInst.File
		if fileName == "" && len(pkg.Downloads) > 0 {
			var url string
			if url, err = m.selectURLForDownload(pkg.Downloads); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to select URL to determine file name: %w"), err)
				return
			}
			fileName = filepath.Base(url)
		}

		var sourcePath string
		if sourcePath, err = m.findSourceFile(fileName, fileInst.Source, downloadedFiles, extractedDirs); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to find source file: %w"), err)
			return
		}

		destination := m.resolveDestination(fileInst.Destination, scopeConfig)

		if err = os.MkdirAll(filepath.Dir(destination), defaultDirMode); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			return
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
			packageBar.SetCurrent(0)
			packageBar.Print()
			if err = m.copyFileWithProgress(actualSourcePath, destination, fileInst.Source, packageBar); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
				return
			}
		}

		var info os.FileInfo
		if info, err = os.Stat(destination); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to get file information: %w"), err)
			return
		}

		if fileInst.Checksum != "" && !skipCopy {
			parts := strings.Split(fileInst.Checksum, ":")
			if len(parts) == 2 {
				if err = m.validationEngine.ValidateChecksum(ctx, destination, parts[0], parts[1]); err != nil {
					err = fmt.Errorf(i18n.Msg("Checksum validation failed: %w"), err)
					return
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
			err = fmt.Errorf(i18n.Msg("Error executing post_install script: %w"), err)
			return
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

			var rawBytes []byte
			var readErr error
			if rawBytes, readErr = os.ReadFile(file.Path); readErr != nil {
				err = fmt.Errorf(i18n.Msg("failed to read WASM file: %w"), readErr)
				return
			}

			var wasmBytes []byte
			if wasmBytes, err = plugin.DecodeTGPBytes(rawBytes); err != nil {
				return
			}

			loggerAdapter := logger.NewSlogAdapter(slog.Default())

			var compilationCache wazero.CompilationCache
			var cacheErr error
			compilationCache, cacheErr = cache.GetCompilationCache(ctx)
			if cacheErr != nil {
				slog.Warn(i18n.Msg("Failed to get compilation cache, continuing without cache"), slog.Any("error", cacheErr))
			}

			var tempHost *host.Host
			var hostErr error
			tgPath := scopeConfig.ConfigDir
			if compilationCache != nil {
				tempHost, hostErr = wasm.New(ctx, wasmBytes, plugin.Info{}, ".", loggerAdapter, wasm.WithCompilationCache(compilationCache), wasm.WithTGPath(tgPath))
			} else {
				tempHost, hostErr = wasm.New(ctx, wasmBytes, plugin.Info{}, ".", loggerAdapter, wasm.WithTGPath(tgPath))
			}
			if hostErr != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "WASM host", hostErr)
				return
			}
			defer wasm.Close(context.Background(), tempHost)

			var info plugin.Info
			var infoErr error
			if info, infoErr = imports.Info(context.Background(), tempHost); infoErr != nil {
				err = fmt.Errorf(i18n.Msg("failed to get plugin info: %w"), infoErr)
				return
			}

			if info.Name == "" {
				err = errors.New(i18n.Msg("invalid plugin: missing name"))
				return
			}

			if info.Name != pkg.Name {
				err = fmt.Errorf("%s: %s != %s", i18n.Msg("plugin name does not match package name"), info.Name, pkg.Name)
				return
			}

			installation.Commands = convertCommandsToModel(info.Commands)
			installation.Options = convertOptionsToModel(info.Options)
			installation.Kind = info.Kind
			installation.Silent = info.Silent
			installation.Always = info.Always
			installation.InitPkgs = info.InitPkgs
			installation.AllowedHosts = info.AllowedHosts
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
		err = errors.New(i18n.Msg("package appears to be a plugin but no .tgp file found"))
		return
	}

	if err = m.databaseManager.RecordInstallation(ctx, installation); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to record installation in database: %w"), err)
		return
	}

	return
}

// generateInstallationID генерирует детерминированный UUIDv5 для установки на основе Package и Version.
// Это гарантирует, что один и тот же пакет с одной версией всегда будет иметь одинаковый ID, независимо от источника.
// Источник не важен - при установке пакета из другого источника с той же версией произойдет замена установки.
func (m *manager) generateInstallationID(packageName string, version string) (id string) {

	name := packageName + packageVersionSep + version
	idUUID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(name))

	return idUUID.String()
}

// GenerateInstallationID генерирует детерминированный ID установки на основе имени пакета и версии.
// Это публичная функция для использования вне пакета installation.
func GenerateInstallationID(packageName string, version string) (id string) {

	name := packageName + packageVersionSep + version
	idUUID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(name))

	return idUUID.String()
}

// findFileInstForDownload возвращает fileInst, соответствующий загружаемому файлу (для одиночного файла, не из архива).
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

// findSourceFile находит исходный файл в загруженных файлах или архивах.
func (m *manager) findSourceFile(fileName string, sourcePath string, downloadedFiles map[string]string, extractedDirs map[string]string) (path string, err error) {

	if extractedDir, exists := extractedDirs[fileName]; exists {
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

	if downloadedFile, exists := downloadedFiles[fileName]; exists {
		if sourcePath != "" {
			path = filepath.Join(filepath.Dir(downloadedFile), sourcePath)
			return
		}
		path = downloadedFile
		return
	}

	return "", fmt.Errorf(i18n.Msg("File %s not found"), fileName)
}

// resolveDestination разрешает путь назначения с учётом scope и переменных.
func (m *manager) resolveDestination(destination string, scopeConfig *storage.ScopeConfig) (path string) {

	destination = strings.ReplaceAll(destination, "${OS}", runtime.GOOS)
	destination = strings.ReplaceAll(destination, "${ARCH}", runtime.GOARCH)

	return filepath.Join(scopeConfig.InstallPrefix, destination)
}

// normalizePackage нормализует пакет: заполняет пустые source в зависимостях source основного пакета.
func (m *manager) normalizePackage(ctx context.Context, pkg *models.Package) (normalized *models.Package) {

	mainSource := ""
	if sourceVal := ctx.Value(contextkeys.Source); sourceVal != nil {
		if source, ok := sourceVal.(string); ok {
			mainSource = source
		}
	}

	if mainSource == "" {
		var err error
		mainSource, err = m.findPackageSource(ctx, pkg)
		if err != nil {
			return pkg
		}
	}

	needsNormalization := false
	for _, depStr := range pkg.Dependencies {
		var parsedURI uri.URI
		var parseErr error
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

	// Создаем копию пакета с нормализованными зависимостями
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
		var parsedURI uri.URI
		var parseErr error
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
			// Оставляем как есть
			normalized.Dependencies[i] = depStr
		}
	}

	return normalized
}

// downloadWithProgress скачивает файл с отображением прогресса на общем прогресс-баре.
func (m *manager) downloadWithProgress(ctx context.Context, url string, destination string, bar *ui.ProgressBar) (err error) {

	if strings.HasPrefix(url, protocolFile) {
		return m.downloadManager.Download(ctx, url, destination)
	}

	progressChan := make(chan int, 10)
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.downloadManager.DownloadWithProgress(ctx, url, destination, progressChan)
	}()

	// Читаем прогресс и обновляем прогресс-бар
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

// packageToInstallInfo представляет информацию о пакете для установки.
type packageToInstallInfo struct {
	pkgName string
	v       models.Version
}

// printDependencyTree выводит дерево зависимостей перед установкой.
func (m *manager) printDependencyTree(ctx context.Context, graph *models.DependencyGraph, packagesInfo []packageToInstallInfo, rootPackageName string, rootVersion models.Version) {

	packagesToInstallMap := make(map[string]models.Version)
	for _, pkgInfo := range packagesInfo {
		packagesToInstallMap[pkgInfo.pkgName] = pkgInfo.v
	}

	var source string
	if sourceVal := ctx.Value(contextkeys.Source); sourceVal != nil {
		if s, ok := sourceVal.(string); ok {
			source = s
		}
	}

	if source == "" {
		var rootPkg *models.Package
		for _, node := range graph.Nodes {
			if node.Package.Name == rootPackageName {
				rootPkg = node.Package
				break
			}
		}
		if rootPkg != nil {
			var err error
			source, err = m.findPackageSource(ctx, rootPkg)
			if err != nil {
				source = noSourcePlaceholder
			}
		} else {
			source = noSourcePlaceholder
		}
	}

	// Строим дерево рекурсивно для пакетов
	var buildPackageTreeNode func(node *models.DependencyNode, visited map[string]bool) pterm.TreeNode
	buildPackageTreeNode = func(node *models.DependencyNode, visited map[string]bool) (treeNode pterm.TreeNode) {
		if visited[node.ID] {
			visitedText := fmt.Sprintf("%s v%s", node.Package.Name, getVersionString(node.Version, rootVersion))
			if node.Package.Descr != "" {
				visitedText += " - " + node.Package.Descr
			}
			return pterm.TreeNode{
				Text: visitedText,
			}
		}
		visited[node.ID] = true

		nodeText := fmt.Sprintf("%s v%s", node.Package.Name, getVersionString(node.Version, rootVersion))

		if node.Package.Descr != "" {
			nodeText += " - " + node.Package.Descr
		}

		if _, willInstall := packagesToInstallMap[node.Package.Name]; !willInstall {
			// Пакет уже установлен, помечаем это зелёной галочкой
			nodeText += " " + pterm.Green("✓")
		}

		// Находим дочерние узлы (зависимости)
		var children []pterm.TreeNode
		for _, edge := range graph.Edges {
			if edge.From.ID == node.ID {
				childNode := edge.To
				if childNode != nil {
					childTreeNode := buildPackageTreeNode(childNode, visited)
					children = append(children, childTreeNode)
				}
			}
		}

		return pterm.TreeNode{
			Text:     nodeText,
			Children: children,
		}
	}

	// Находим все пакеты из этого источника (корневой пакет и его зависимости)
	var packageNodes []pterm.TreeNode
	visited := make(map[string]bool)

	// Находим корневой узел и строим дерево от него
	var rootNode *models.DependencyNode
	for _, node := range graph.Nodes {
		if node.Package.Name == rootPackageName {
			rootNode = node
			break
		}
	}

	if rootNode != nil {
		packageTreeNode := buildPackageTreeNode(rootNode, visited)
		packageNodes = append(packageNodes, packageTreeNode)
	}

	rootTreeNode := pterm.TreeNode{
		Text:     source,
		Children: packageNodes,
	}

	pterm.Println()
	if err := pterm.DefaultTree.WithRoot(rootTreeNode).Render(); err != nil {
		slog.Debug(i18n.Msg("Failed to render dependency tree"), slog.Any("error", err))
		return
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
