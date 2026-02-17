// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"

	"github.com/tetratelabs/wazero"
)

const (
	accessLevelRead  = "r"
	accessLevelWrite = "w"
)

// ResolvedMount — результат однократного разрешения пути: используется и для монтирования, и для лога.
type ResolvedMount struct {
	PathKey      string
	ExpandedPath string
	MountPoint   string
	AccessLevel  string
}

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

func (b *Builder) Build(info plugin.Info) (fsConfig wazero.FSConfig, resolved []ResolvedMount, err error) {

	if len(info.AllowedPaths) == 0 {
		return wazero.NewFSConfig(), nil, nil
	}

	resolved = resolveMounts(b.rootDir, b.tgPath, info)
	fsConfig = b.buildFromResolved(resolved)
	return
}

func resolveMounts(rootDir string, tgPath string, info plugin.Info) (out []ResolvedMount) {

	mapper := newMountPointMapper()
	expander := newPathExpander(rootDir, tgPath)
	paths := filterPathsByGoPriority(info.AllowedPaths)
	out = make([]ResolvedMount, 0, len(paths))

	for pathKey, accessLevel := range paths {
		expandedPath, expandErr := expander.expand(pathKey)
		if expandErr != nil || expandedPath == "" {
			continue
		}
		pathType := detectPathType(pathKey)
		mountPoint := mapper.mapMountPoint(pathKey, expandedPath, pathType)
		accessLevel = normalizeAccessLevel(accessLevel)
		if accessLevel != accessLevelRead && accessLevel != accessLevelWrite {
			continue
		}
		out = append(out, ResolvedMount{
			PathKey:      pathKey,
			ExpandedPath: expandedPath,
			MountPoint:   mountPoint,
			AccessLevel:  accessLevel,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].MountPoint < out[j].MountPoint
	})

	return out
}

func (b *Builder) buildFromResolved(resolved []ResolvedMount) (fsConfig wazero.FSConfig) {

	fsConfig = wazero.NewFSConfig()

	for _, r := range resolved {
		switch r.AccessLevel {
		case accessLevelRead:
			if err := b.directoryExists(r.ExpandedPath); err != nil {
				slog.Warn(i18n.Msg("Read-only mount directory does not exist, skipping"), "path", r.PathKey, "expandedPath", r.ExpandedPath, "error", err)
				continue
			}
			fsConfig = fsConfig.WithReadOnlyDirMount(r.ExpandedPath, r.MountPoint)
		case accessLevelWrite:
			if err := b.ensureDirectoryExists(r.ExpandedPath); err != nil {
				slog.Warn(fmt.Sprintf(i18n.Msg("Failed to create %s"), "directory for mount"), "path", r.PathKey, "expandedPath", r.ExpandedPath, "error", err)
				continue
			}
			fsConfig = fsConfig.WithDirMount(r.ExpandedPath, r.MountPoint)
		default:
			slog.Warn(i18n.Msg("Unknown access level, skipping mount"), "path", r.PathKey, "accessLevel", r.AccessLevel)
		}
	}

	return
}

// normalizeAccessLevel нормализует уровень доступа.
func normalizeAccessLevel(level string) (normalized string) {

	return strings.ToLower(strings.TrimSpace(level))
}

func (b *Builder) directoryExists(path string) (err error) {

	cleanPath := filepath.Clean(path)

	var fileInfo os.FileInfo
	if fileInfo, err = os.Stat(cleanPath); err != nil {
		return
	}
	if !fileInfo.IsDir() {
		return os.ErrExist
	}
	return
}

func (b *Builder) ensureDirectoryExists(path string) (err error) {

	cleanPath := filepath.Clean(path)

	var fileInfo os.FileInfo
	if fileInfo, err = os.Stat(cleanPath); err == nil {
		if !fileInfo.IsDir() {
			return os.ErrExist
		}
		return
	}

	if !os.IsNotExist(err) {
		return
	}

	return os.MkdirAll(cleanPath, 0700)
}

// filterPathsByGoPriority фильтрует пути: если есть @go, удаляет все @root.
func filterPathsByGoPriority(paths map[string]string) (filtered map[string]string) {

	hasGo := false
	for pathKey := range paths {
		if detectPathType(pathKey) == PathTypeGo {
			hasGo = true
			break
		}
	}

	if !hasGo {
		return paths
	}

	filtered = make(map[string]string, len(paths))
	for pathKey, accessLevel := range paths {
		if detectPathType(pathKey) != PathTypeRoot {
			filtered[pathKey] = accessLevel
		}
	}

	return
}
