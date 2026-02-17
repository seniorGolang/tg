// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package version

import (
	"golang.org/x/mod/semver"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func IsPreRelease(v models.Version) (isPreRelease bool) {

	versionStr := normalizeVersionString(v.Original)
	if versionStr == "" {
		return false
	}

	return semver.Prerelease(versionStr) != ""
}
