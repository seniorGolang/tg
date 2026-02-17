// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"fmt"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"

	"golang.org/x/mod/semver"
)

const (
	versionPrefix = "v"
)

// parseTag парсит тег и извлекает версию.
// Поддерживает только формат: v{version}
func parseTag(tag string) (version string, err error) {

	if !strings.HasPrefix(tag, VersionTagPrefix) {
		return "", fmt.Errorf(i18n.Msg("Tag must start with '%s'"), VersionTagPrefix)
	}

	version = strings.TrimPrefix(tag, VersionTagPrefix)
	if !semver.IsValid(VersionTagPrefix + version) {
		return "", fmt.Errorf(i18n.Msg("Invalid SemVer version: %s"), version)
	}

	return
}

// compareVersions сравнивает две версии в формате SemVer.
// Возвращает: >0 если v1 > v2, <0 если v1 < v2, 0 если v1 == v2
func compareVersions(v1, v2 string) int {

	v1Normalized := strings.TrimPrefix(v1, versionPrefix)
	v2Normalized := strings.TrimPrefix(v2, versionPrefix)

	v1WithPrefix := versionPrefix + v1Normalized
	v2WithPrefix := versionPrefix + v2Normalized

	return semver.Compare(v1WithPrefix, v2WithPrefix)
}
