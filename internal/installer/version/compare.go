// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package version

import (
	"strings"

	"golang.org/x/mod/semver"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// Compare сравнивает две версии.
// Возвращает:
//   - 1, если v1 > v2
//   - -1, если v1 < v2
//   - 0, если v1 == v2
func Compare(v1 models.Version, v2 models.Version) (result int) {

	v1Str := normalizeVersionString(v1.Original)
	v2Str := normalizeVersionString(v2.Original)

	return semver.Compare(v1Str, v2Str)
}

// normalizeVersionString нормализует строку версии для использования с semver.
// golang.org/x/mod/semver ожидает префикс "v" (например, "v1.0.0"); без него Compare может работать некорректно.
func normalizeVersionString(version string) (normalized string) {

	if version == "" {
		return ""
	}

	version = strings.TrimSpace(version)

	if !strings.HasPrefix(version, VersionPrefix) {
		version = VersionPrefix + version
	}

	if !semver.IsValid(version) {
		return version
	}

	return semver.Canonical(version)
}
