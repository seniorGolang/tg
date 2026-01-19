// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// SafeJoin безопасно объединяет пути, нормализуя их и проверяя на path traversal атаки.
func SafeJoin(basePath string, paths ...string) (safePath string, err error) {

	basePath = filepath.Clean(basePath)
	fullPath := filepath.Join(append([]string{basePath}, paths...)...)
	fullPath = filepath.Clean(fullPath)

	var relPath string
	if relPath, err = filepath.Rel(basePath, fullPath); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to compute relative path: %w"), err)
	}

	if strings.HasPrefix(relPath, pathTraversalPrefix) {
		return "", errors.New(i18n.Msg("Path traversal attempt detected: path is outside base directory"))
	}

	safePath = fullPath
	return
}

func SafeBase(path string) (base string) {

	return filepath.Base(filepath.Clean(path))
}

func SafeDir(path string) (dir string) {

	return filepath.Dir(filepath.Clean(path))
}

func ValidatePath(path string) (err error) {

	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, pathTraversalPrefix) {
		return fmt.Errorf(i18n.Msg("Path contains path traversal attempt: %s"), path)
	}
	return
}
