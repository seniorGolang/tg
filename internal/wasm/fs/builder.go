// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"

	"github.com/tetratelabs/wazero"
)

const (
	// accessLevelRead - уровень доступа только для чтения.
	accessLevelRead = "r"
	// accessLevelWrite - уровень доступа для записи.
	accessLevelWrite = "w"
)

type Builder struct {
	rootDir  string
	tgPath   string
	expander *pathExpander
	mapper   *mountPointMapper
}

func NewBuilder(rootDir string, tgPath string) (builder *Builder) {

	return &Builder{
		rootDir:  rootDir,
		tgPath:   tgPath,
		expander: newPathExpander(rootDir, tgPath),
		mapper:   newMountPointMapper(),
	}
}

func (b *Builder) Build(info plugin.Info) (fsConfig wazero.FSConfig, err error) {

	// Если AllowedPaths пустой, не предоставляем доступ к файловой системе
	if len(info.AllowedPaths) == 0 {
		fsConfig = wazero.NewFSConfig()
		return
	}

	return b.BuildFromPaths(info.AllowedPaths, false)
}

// BuildFromPaths: при mountRootDir монтирует rootDir в корень WASM ФС.
func (b *Builder) BuildFromPaths(paths map[string]string, mountRootDir bool) (fsConfig wazero.FSConfig, err error) {

	fsConfig = wazero.NewFSConfig()

	// Монтируем rootDir в корень WASM файловой системы, если нужно и rootDir не пустой
	if mountRootDir && b.rootDir != "" {
		fsConfig = fsConfig.WithDirMount(b.rootDir, "/")
	}

	// Обрабатываем paths, если они указаны
	if len(paths) == 0 {
		return
	}

	// Если есть @go, удаляем все @root - @go имеет приоритет
	paths = filterPathsByGoPriority(paths)

	for pathKey, accessLevel := range paths {
		expandedPath, expandErr := b.expander.expand(pathKey)
		if expandErr != nil {
			continue
		}

		if expandedPath == "" {
			continue
		}

		pathType := DetectPathType(pathKey)
		mountPoint := b.mapper.mapMountPoint(pathKey, expandedPath, pathType)

		accessLevel = normalizeAccessLevel(accessLevel)

		switch accessLevel {
		case accessLevelRead:
			fsConfig = fsConfig.WithReadOnlyDirMount(expandedPath, mountPoint)
		case accessLevelWrite:
			// Для путей с правами записи создаём директорию, если её нет
			if err := b.ensureDirectoryExists(expandedPath); err != nil {
				slog.Warn(fmt.Sprintf(i18n.Msg("Failed to create %s"), "directory for mount"),
					"path", pathKey,
					"expandedPath", expandedPath,
					"error", err)
				continue
			}
			fsConfig = fsConfig.WithDirMount(expandedPath, mountPoint)
		default:
			slog.Warn(i18n.Msg("Unknown access level, skipping mount"),
				"path", pathKey,
				"accessLevel", accessLevel)
			continue
		}
	}

	return
}

// normalizeAccessLevel нормализует уровень доступа.
func normalizeAccessLevel(level string) (normalized string) {

	normalized = strings.ToLower(strings.TrimSpace(level))
	return
}

func (b *Builder) ensureDirectoryExists(path string) (err error) {

	cleanPath := filepath.Clean(path)

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(cleanPath)
	if err == nil {
		if !fileInfo.IsDir() {
			return os.ErrExist
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	if mkdirErr := os.MkdirAll(cleanPath, 0755); mkdirErr != nil {
		return mkdirErr
	}

	return nil
}

// filterPathsByGoPriority фильтрует пути: если есть @go, удаляет все @root.
func filterPathsByGoPriority(paths map[string]string) (filtered map[string]string) {

	hasGo := false
	for pathKey := range paths {
		if DetectPathType(pathKey) == PathTypeGo {
			hasGo = true
			break
		}
	}

	if !hasGo {
		return paths
	}

	filtered = make(map[string]string, len(paths))
	for pathKey, accessLevel := range paths {
		if DetectPathType(pathKey) != PathTypeRoot {
			filtered[pathKey] = accessLevel
		}
	}

	return
}
