// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package version

import (
	"strings"

	"golang.org/x/mod/semver"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// Match: операторы >=, <=, >, <, ~, ^ или точное совпадение.
func Match(constraint string, v models.Version) (matches bool) {

	if constraint == "" {
		return true
	}

	constraint = strings.TrimSpace(constraint)

	if constraint == LatestVersion {
		return true
	}

	versionStr := normalizeVersionString(v.Original)
	if versionStr == "" {
		return false
	}

	if strings.HasPrefix(constraint, ">=") {
		constraintStr := normalizeVersionString(strings.TrimPrefix(constraint, ">="))
		if constraintStr == "" || !semver.IsValid(constraintStr) {
			return false
		}
		return semver.Compare(versionStr, constraintStr) >= 0
	}

	if strings.HasPrefix(constraint, "<=") {
		constraintStr := normalizeVersionString(strings.TrimPrefix(constraint, "<="))
		if constraintStr == "" || !semver.IsValid(constraintStr) {
			return false
		}
		return semver.Compare(versionStr, constraintStr) <= 0
	}

	if strings.HasPrefix(constraint, ">") {
		constraintStr := normalizeVersionString(strings.TrimPrefix(constraint, ">"))
		if constraintStr == "" || !semver.IsValid(constraintStr) {
			return false
		}
		return semver.Compare(versionStr, constraintStr) > 0
	}

	if strings.HasPrefix(constraint, "<") {
		constraintStr := normalizeVersionString(strings.TrimPrefix(constraint, "<"))
		if constraintStr == "" || !semver.IsValid(constraintStr) {
			return false
		}
		return semver.Compare(versionStr, constraintStr) < 0
	}

	if strings.HasPrefix(constraint, "~") {
		constraintStr := normalizeVersionString(strings.TrimPrefix(constraint, "~"))
		if constraintStr == "" || !semver.IsValid(constraintStr) {
			return false
		}
		majorMinorV := semver.MajorMinor(versionStr)
		majorMinorC := semver.MajorMinor(constraintStr)
		if majorMinorV != majorMinorC {
			return false
		}
		return semver.Compare(versionStr, constraintStr) >= 0
	}

	if strings.HasPrefix(constraint, "^") {
		constraintStr := normalizeVersionString(strings.TrimPrefix(constraint, "^"))
		if constraintStr == "" || !semver.IsValid(constraintStr) {
			return false
		}
		majorV := semver.Major(versionStr)
		majorC := semver.Major(constraintStr)
		if majorV != majorC {
			return false
		}
		return semver.Compare(versionStr, constraintStr) >= 0
	}

	constraintStr := normalizeVersionString(constraint)
	if constraintStr == "" || !semver.IsValid(constraintStr) {
		return false
	}

	return semver.Compare(versionStr, constraintStr) == 0
}
