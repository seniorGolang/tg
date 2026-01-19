// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import (
	"path/filepath"
	"strings"
)

// mountPointMapper определяет mount point в WASM файловой системе для пути.
type mountPointMapper struct{}

func newMountPointMapper() (mapper *mountPointMapper) {

	return &mountPointMapper{}
}

// mapMountPoint определяет mount point в WASM файловой системе для пути.
func (m *mountPointMapper) mapMountPoint(pathKey string, expandedPath string, pathType PathType) (mountPoint string) {

	switch pathType {
	case PathTypeGo:
		// @go -> /
		// @go/ -> /
		// @go/internal/cli -> /internal/cli
		trimmed := strings.TrimPrefix(pathKey, pathPrefixGo)
		if trimmed == "" || trimmed == "/" {
			mountPoint = "/"
			return
		}
		relativePath := strings.TrimPrefix(trimmed, "/")
		mountPoint = "/" + filepath.ToSlash(filepath.Clean(relativePath))
		return

	case PathTypeRoot:
		// @root -> /
		// @root/ -> /
		// @root/internal/cli -> /internal/cli
		trimmed := strings.TrimPrefix(pathKey, pathPrefixRoot)
		if trimmed == "" || trimmed == "/" {
			mountPoint = "/"
			return
		}
		relativePath := strings.TrimPrefix(trimmed, "/")
		mountPoint = "/" + filepath.ToSlash(filepath.Clean(relativePath))
		return

	case PathTypeTG:
		// @tg -> /tg
		// @tg/ -> /tg/
		// @tg/myplugin -> /tg/myplugin
		trimmed := strings.TrimPrefix(pathKey, pathPrefixTG)
		if trimmed == "" {
			mountPoint = "/tg"
			return
		}
		if trimmed == "/" {
			mountPoint = "/tg/"
			return
		}
		relativePath := strings.TrimPrefix(trimmed, "/")
		mountPoint = "/tg/" + filepath.ToSlash(filepath.Clean(relativePath))
		return

	case PathTypeEnv, PathTypeHome, PathTypeAbsolute:
		// Для абсолютных путей и путей с переменными используем расширенный путь
		normalized := filepath.ToSlash(filepath.Clean(expandedPath))
		if !strings.HasPrefix(normalized, "/") {
			normalized = "/" + normalized
		}
		mountPoint = normalized
		return

	default:
		mountPoint = "/" + filepath.ToSlash(filepath.Clean(expandedPath))
		return
	}
}
