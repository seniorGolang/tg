// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package version

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// Parse парсит строку версии в структуру Version.
func Parse(version string) (v models.Version, err error) {

	original := version

	normalized := normalizeVersionString(original)
	if normalized == "" || !semver.IsValid(normalized) {
		err = fmt.Errorf(i18n.Msg("Invalid version format: %s"), original)
		return
	}

	canonical := semver.Canonical(normalized)
	prerelease := semver.Prerelease(normalized)
	build := semver.Build(normalized)

	var major, minor, patch int
	if major, minor, patch, err = extractVersionNumbers(canonical); err != nil {
		err = fmt.Errorf(i18n.Msg("Error parsing version numbers: %w"), err)
		return
	}

	preReleaseStr := ""
	if prerelease != "" {
		preReleaseStr = strings.TrimPrefix(prerelease, "-")
	}

	buildStr := ""
	if build != "" {
		buildStr = strings.TrimPrefix(build, "+")
	}

	v = models.Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		PreRelease: preReleaseStr,
		Build:      buildStr,
		Original:   original,
	}
	return
}

// extractVersionNumbers извлекает числовые значения major, minor, patch из канонической строки версии.
func extractVersionNumbers(canonical string) (major int, minor int, patch int, err error) {

	if !strings.HasPrefix(canonical, VersionPrefix) {
		err = fmt.Errorf("version must start with %s", VersionPrefix)
		return
	}

	versionWithoutPrefix := strings.TrimPrefix(canonical, VersionPrefix)
	parts := strings.Split(versionWithoutPrefix, ".")
	if len(parts) < 1 || len(parts) > 3 {
		err = fmt.Errorf("invalid version format")
		return
	}

	if major, err = strconv.Atoi(parts[0]); err != nil {
		err = fmt.Errorf("error parsing major version: %w", err)
		return
	}

	if len(parts) > 1 {
		if minor, err = strconv.Atoi(parts[1]); err != nil {
			err = fmt.Errorf("error parsing minor version: %w", err)
			return
		}
	}

	if len(parts) > 2 {
		patchPart := parts[2]
		if idx := strings.Index(patchPart, "-"); idx != -1 {
			patchPart = patchPart[:idx]
		}
		if patch, err = strconv.Atoi(patchPart); err != nil {
			err = fmt.Errorf("error parsing patch version: %w", err)
			return
		}
	}

	return
}
