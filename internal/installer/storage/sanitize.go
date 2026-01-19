// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"path/filepath"
	"strings"
)

// removeFileNameFromPath убирает имя файла (manifest.yml, manifest.yaml) из конца пути.
func removeFileNameFromPath(path string) (cleanedPath string) {

	cleanedPath = strings.TrimSuffix(path, PathSeparator+ManifestFileName)
	cleanedPath = strings.TrimSuffix(cleanedPath, PathSeparator+ManifestFileNameYAML)
	cleanedPath = strings.TrimSuffix(cleanedPath, ManifestFileName)
	cleanedPath = strings.TrimSuffix(cleanedPath, ManifestFileNameYAML)

	return cleanedPath
}

// sanitizePathComponent очищает компонент пути от небезопасных символов.
func sanitizePathComponent(component string) (sanitized string) {

	sanitized = strings.Map(func(r rune) rune {
		if unsafeChars[r] {
			return '_'
		}
		return r
	}, component)
	return
}

// sanitizePath очищает путь от небезопасных символов, сохраняя структуру папок.
func sanitizePath(path string) (sanitized string) {

	parts := strings.Split(path, PathSeparator)
	sanitizedParts := make([]string, 0, len(parts))

	for _, part := range parts {
		if part != "" {
			sanitizedParts = append(sanitizedParts, sanitizePathComponent(part))
		}
	}

	return filepath.Join(sanitizedParts...)
}
