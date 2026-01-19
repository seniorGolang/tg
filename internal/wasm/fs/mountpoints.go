// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import "github.com/seniorGolang/tg/v3/internal/plugin"

// MountPointInfo содержит информацию о mount point.
type MountPointInfo struct {
	PathKey     string `json:"pathKey"`
	MountPoint  string `json:"mountPoint"`
	AccessLevel string `json:"accessLevel"`
}

func GetMountPoints(rootDir string, tgPath string, info plugin.Info) (mountPoints []MountPointInfo) {

	if len(info.AllowedPaths) == 0 {
		return nil
	}

	expander := newPathExpander(rootDir, tgPath)
	mapper := newMountPointMapper()

	// Если есть @go, удаляем все @root - @go имеет приоритет
	filteredPaths := filterPathsByGoPriority(info.AllowedPaths)

	mountPoints = make([]MountPointInfo, 0, len(filteredPaths))

	for pathKey, accessLevel := range filteredPaths {
		expandedPath, expandErr := expander.expand(pathKey)
		if expandErr != nil {
			continue
		}

		if expandedPath == "" {
			continue
		}

		pathType := DetectPathType(pathKey)
		mountPoint := mapper.mapMountPoint(pathKey, expandedPath, pathType)

		accessLevel = normalizeAccessLevel(accessLevel)

		// Проверяем, что уровень доступа валидный
		if accessLevel != accessLevelRead && accessLevel != accessLevelWrite {
			continue
		}

		mountPoints = append(mountPoints, MountPointInfo{
			PathKey:     pathKey,
			MountPoint:  mountPoint,
			AccessLevel: accessLevel,
		})
	}

	return
}
