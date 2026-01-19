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
				// Проверяем, являются ли манифесты обновлениями одного источника
				manifestURLs := make([]string, 0, len(uniqueManifests))
				for manifestURL := range uniqueManifests {
					manifestURLs = append(manifestURLs, manifestURL)
				}

				// Извлекаем версии из путей манифестов
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

				// Если все версии валидны и они из одного источника - проверяем, является ли это обновлением
				if allVersionsValid && len(versions) == len(manifestURLs) {
					// Проверяем, что все манифесты из одного источника
					sources := make(map[string]bool)
					for _, info := range infos {
						sources[info.Source] = true
					}

					// Если все из одного источника и версии можно сравнить - это обновление, а не конфликт
					if len(sources) == 1 {
						// Собираем все версии
						versionList := make([]string, 0, len(versions))
						for _, v := range versions {
							versionList = append(versionList, v)
						}

						// Если все версии можно сравнить между собой - это обновление
						if m.areVersionsComparable(versionList) {
							// Это обновление, пропускаем
							continue
						}
					}
				}

				// Это конфликт
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

// extractSourceFromURL извлекает источник из нормализованного URL.
func (m *manager) extractSourceFromURL(normalizedURL string) (source string) {

	// Проверяем, является ли это старым форматом (содержит подчеркивания и не содержит слешей)
	if !strings.Contains(normalizedURL, string(filepath.Separator)) && strings.Contains(normalizedURL, "_") {
		// Старый формат
		parts := strings.Split(normalizedURL, "_")
		if len(parts) >= 3 {
			return strings.Join(parts[:len(parts)-1], "/")
		}
		return normalizedURL
	}

	// Новый формат - путь с папками
	// Используем функцию из storage
	return storage.ExtractSourceFromNormalizedPath(normalizedURL)
}

// extractVersionFromManifestURL извлекает версию из пути манифеста.
// Например, из "github.com/seniorGolang/tg-plugins/releases/download/v2.4.14" извлекает "v2.4.14" или "2.4.14".
func (m *manager) extractVersionFromManifestURL(manifestURL string) (version string) {

	// Ищем паттерн /releases/download/v{version}
	if strings.Contains(manifestURL, releasesDownloadPath) {
		parts := strings.Split(manifestURL, releasesDownloadPath)
		if len(parts) == 2 {
			versionPart := parts[1]
			// Убираем возможные дополнительные части пути после версии
			versionPart = strings.Split(versionPart, "/")[0]
			// Проверяем, что это похоже на версию (начинается с v или является SemVer)
			if strings.HasPrefix(versionPart, ver.VersionPrefix) {
				// Проверяем валидность версии через Parse
				if _, parseErr := ver.Parse(versionPart); parseErr == nil {
					return versionPart
				}
			} else {
				// Пробуем добавить префикс v
				versionWithPrefix := ver.VersionPrefix + versionPart
				if _, parseErr := ver.Parse(versionWithPrefix); parseErr == nil {
					return versionWithPrefix
				}
			}
		}
	}

	// Если не нашли паттерн releases/download, пробуем извлечь из последней части пути
	parts := strings.Split(manifestURL, string(filepath.Separator))
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Проверяем, похоже ли это на версию
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

	// Парсим все версии и проверяем их валидность
	parsedVersions := make([]models.Version, 0, len(versions))
	for _, v := range versions {
		parsedVersion, parseErr := ver.Parse(v)
		if parseErr != nil {
			return false
		}
		parsedVersions = append(parsedVersions, parsedVersion)
	}

	// Проверяем, что все версии можно сравнить между собой
	// Достаточно проверить, что все версии валидны (уже проверили выше)
	// и что они не равны друг другу (если равны - это конфликт)
	for i := 0; i < len(parsedVersions); i++ {
		for j := i + 1; j < len(parsedVersions); j++ {
			comparison := ver.Compare(parsedVersions[i], parsedVersions[j])
			if comparison == 0 {
				// Одинаковые версии - это конфликт
				return false
			}
		}
	}

	// Все версии валидны и различны - это обновления
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
