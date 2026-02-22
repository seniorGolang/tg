// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package scope

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/manifest"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	ver "github.com/seniorGolang/tg/v3/internal/installer/version"
)

const (
	conflictTypePackageDuplicate = "package_duplicate"
	conflictTypeFileConflict     = "file_conflict"
	releasesDownloadPath         = "/releases/download/"
)

func (m *manager) CheckConsistency(ctx context.Context, scopeName string) (err error) {

	catalogDir := storage.GetCatalogDir(scopeName)
	if _, statErr := os.Stat(catalogDir); os.IsNotExist(statErr) {
		return
	}

	manifestMgr := manifest.NewManager(scopeName)

	var allManifests []managers.ManifestWithSource
	if allManifests, err = manifestMgr.GetAllManifests(ctx); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get manifests: %w"), err)
		return
	}

	packageMap := make(map[string][]manifestPackageInfo)
	fileMap := make(map[string][]packageFileInfo)

	for _, manifestWithSource := range allManifests {
		relPath, relErr := filepath.Rel(catalogDir, filepath.Dir(manifestWithSource.Path))
		if relErr != nil {
			continue
		}

		manifestURL := relPath
		source := m.extractSourceFromURL(manifestURL)

		for _, pkg := range manifestWithSource.Manifest.Packages {
			packageID := source + "/" + pkg.Name
			packageMap[packageID] = append(packageMap[packageID], manifestPackageInfo{
				ManifestURL: manifestURL,
				PackageName: pkg.Name,
				Source:      source,
			})

			for _, file := range pkg.Files {
				fileMap[file.Destination] = append(fileMap[file.Destination], packageFileInfo{
					PackageID:   packageID,
					ManifestURL: manifestURL,
					Destination: file.Destination,
				})
			}
		}
	}

	conflicts := make([]consistencyConflict, 0)

	for packageID, infos := range packageMap {
		if len(infos) > 1 {
			uniqueManifests := make(map[string]bool)
			for _, info := range infos {
				uniqueManifests[info.ManifestURL] = true
			}
			if len(uniqueManifests) > 1 {
				manifestURLs := make([]string, 0, len(uniqueManifests))
				for manifestURL := range uniqueManifests {
					manifestURLs = append(manifestURLs, manifestURL)
				}

				versions := make(map[string]string) // manifestURL -> version
				allVersionsValid := true
				for _, manifestURL := range manifestURLs {
					version := m.extractVersionFromManifestURL(manifestURL)
					if version == "" {
						allVersionsValid = false
						break
					}
					versions[manifestURL] = version
				}

				if allVersionsValid && len(versions) == len(manifestURLs) {
					sources := make(map[string]bool)
					for _, info := range infos {
						sources[info.Source] = true
					}

					if len(sources) == 1 {
						versionList := make([]string, 0, len(versions))
						for _, v := range versions {
							versionList = append(versionList, v)
						}

						if m.areVersionsComparable(versionList) {
							continue
						}
					}
				}

				conflicts = append(conflicts, consistencyConflict{
					Type:         conflictTypePackageDuplicate,
					PackageID:    packageID,
					ManifestURLs: manifestURLs,
				})
			}
		}
	}

	for destination, infos := range fileMap {
		if len(infos) > 1 {
			uniquePackages := make(map[string]bool)
			for _, info := range infos {
				uniquePackages[info.PackageID] = true
			}
			if len(uniquePackages) > 1 {
				conflicts = append(conflicts, consistencyConflict{
					Type:        conflictTypeFileConflict,
					Destination: destination,
					PackageIDs:  make([]string, 0, len(uniquePackages)),
				})
				for packageID := range uniquePackages {
					conflicts[len(conflicts)-1].PackageIDs = append(conflicts[len(conflicts)-1].PackageIDs, packageID)
				}
			}
		}
	}

	if len(conflicts) > 0 {
		slog.Error(i18n.Msg("Consistency conflicts detected"), slog.Int("count", len(conflicts)))

		for i, conflict := range conflicts {
			switch conflict.Type {
			case conflictTypePackageDuplicate:
				attrs := []slog.Attr{
					slog.Int("index", i+1),
					slog.String("package", conflict.PackageID),
					slog.Int("manifest_count", len(conflict.ManifestURLs)),
				}
				for j, manifestURL := range conflict.ManifestURLs {
					attrs = append(attrs, slog.String(fmt.Sprintf("manifest.%d", j+1), manifestURL))
				}
				args := make([]any, len(attrs))
				for idx, attr := range attrs {
					args[idx] = attr
				}
				slog.Error(i18n.Msg("Package duplicate"), args...)
			case conflictTypeFileConflict:
				attrs := []slog.Attr{
					slog.Int("index", i+1),
					slog.String("destination", conflict.Destination),
					slog.Int("package_count", len(conflict.PackageIDs)),
				}
				for j, packageID := range conflict.PackageIDs {
					attrs = append(attrs, slog.String(fmt.Sprintf("package.%d", j+1), packageID))
				}
				args := make([]any, len(attrs))
				for idx, attr := range attrs {
					args[idx] = attr
				}
				slog.Error(i18n.Msg("File conflict"), args...)
			default:
				slog.Error(i18n.Msg("Unknown conflict type"), slog.Int("index", i+1), slog.String("type", conflict.Type))
			}
		}

		err = fmt.Errorf(i18n.Msg("Consistency conflicts detected: %d conflicts"), len(conflicts))
		return
	}

	return
}
func (m *manager) extractSourceFromURL(normalizedURL string) (source string) {

	if !strings.Contains(normalizedURL, string(filepath.Separator)) && strings.Contains(normalizedURL, "_") {
		// Старый формат
		parts := strings.Split(normalizedURL, "_")
		if len(parts) >= 3 {
			return strings.Join(parts[:len(parts)-1], "/")
		}
		return normalizedURL
	}

	return storage.ExtractSourceFromNormalizedPath(normalizedURL)
}

// extractVersionFromManifestURL извлекает версию из пути манифеста.
// Например, из "github.com/seniorGolang/tg-plugins/releases/download/v2.4.14" извлекает "v2.4.14" или "2.4.14".
func (m *manager) extractVersionFromManifestURL(manifestURL string) (version string) {

	if strings.Contains(manifestURL, releasesDownloadPath) {
		splitParts := strings.Split(manifestURL, releasesDownloadPath)
		if len(splitParts) == 2 {
			versionPart := splitParts[1]
			versionPart = strings.Split(versionPart, "/")[0]
			if strings.HasPrefix(versionPart, ver.VersionPrefix) {
				if _, parseErr := ver.Parse(versionPart); parseErr == nil {
					return versionPart
				}
			} else {
				versionWithPrefix := ver.VersionPrefix + versionPart
				if _, parseErr := ver.Parse(versionWithPrefix); parseErr == nil {
					return versionWithPrefix
				}
			}
		}
	}

	// Если не нашли паттерн releases/download, пробуем извлечь из последней части пути
	pathParts := strings.Split(manifestURL, string(filepath.Separator))
	if len(pathParts) > 0 {
		lastPart := pathParts[len(pathParts)-1]
		if strings.HasPrefix(lastPart, ver.VersionPrefix) {
			if _, parseErr := ver.Parse(lastPart); parseErr == nil {
				return lastPart
			}
		} else {
			versionWithPrefix := ver.VersionPrefix + lastPart
			if _, parseErr := ver.Parse(versionWithPrefix); parseErr == nil {
				return versionWithPrefix
			}
		}
	}

	return ""
}

// areVersionsComparable: если все версии сравнимы — это обновления, иначе конфликт.
func (m *manager) areVersionsComparable(versions []string) (areComparable bool) {

	if len(versions) < 2 {
		return true
	}

	parsedVersions := make([]models.Version, 0, len(versions))
	for _, v := range versions {
		parsedVersion, parseErr := ver.Parse(v)
		if parseErr != nil {
			return false
		}
		parsedVersions = append(parsedVersions, parsedVersion)
	}

	for i := 0; i < len(parsedVersions); i++ {
		for j := i + 1; j < len(parsedVersions); j++ {
			comparison := ver.Compare(parsedVersions[i], parsedVersions[j])
			if comparison == 0 {
				return false
			}
		}
	}

	return true
}

// manifestPackageInfo содержит информацию о пакете в манифесте.
type manifestPackageInfo struct {
	ManifestURL string
	PackageName string
	Source      string
}

// packageFileInfo содержит информацию о файле пакета.
type packageFileInfo struct {
	PackageID   string
	ManifestURL string
	Destination string
}

// consistencyConflict представляет конфликт непротиворечивости.
type consistencyConflict struct {
	Type         string
	PackageID    string
	Destination  string
	ManifestURLs []string
	PackageIDs   []string
}
