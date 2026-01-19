// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func ParseDependency(dep string) (name string, version string, err error) {

	depModel := models.ParseDependencyString(dep)
	name = depModel.Package
	version = depModel.Version

	return
}

func normalizeVersion(version string) (normalized string) {

	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, versionPrefixV) {
		return version
	}
	return versionPrefixV + version
}

func isVersionCompatible(installedVersion string, requirement string) (compatible bool) {

	if requirement == "" {
		return true
	}

	installedVersionNormalized := normalizeVersion(installedVersion)
	if !semver.IsValid(installedVersionNormalized) {
		return false
	}

	requirement = strings.TrimSpace(requirement)

	switch {
	case strings.HasPrefix(requirement, requirementPrefixCaret):
		return checkCaretRequirement(installedVersionNormalized, requirement)
	case strings.HasPrefix(requirement, requirementPrefixTilde):
		return checkTildeRequirement(installedVersionNormalized, requirement)
	case strings.HasPrefix(requirement, requirementPrefixGreaterEqual):
		return checkGreaterEqualRequirement(installedVersionNormalized, requirement)
	case strings.HasPrefix(requirement, requirementPrefixLessEqual):
		return checkLessEqualRequirement(installedVersionNormalized, requirement)
	case strings.HasPrefix(requirement, requirementPrefixGreater):
		return checkGreaterRequirement(installedVersionNormalized, requirement)
	case strings.HasPrefix(requirement, requirementPrefixLess):
		return checkLessRequirement(installedVersionNormalized, requirement)
	case strings.HasPrefix(requirement, requirementPrefixEqual):
		return checkEqualRequirement(installedVersionNormalized, requirement)
	default:
		return checkExactVersion(installedVersionNormalized, requirement)
	}
}

// checkCaretRequirement: по semver ^1.2.3 => [1.2.3, 2.0.0); ^0.1.2 => [0.1.2, 0.2.0).
func checkCaretRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixCaret))
	if !semver.IsValid(baseVersion) {
		return false
	}

	major := semver.Major(baseVersion)
	majorInt := parseMajorVersion(major)

	// Для major=0 семантика caret: только minor не ломается (0.1.x в рамках 0.1.*).
	if majorInt == 0 {
		minor := semver.MajorMinor(baseVersion)
		if minor == "" {
			return semver.Compare(installedVersion, baseVersion) >= 0
		}
		nextMinor := incrementMinorVersion(minor)
		if nextMinor == "" {
			return false
		}
		return semver.Compare(installedVersion, baseVersion) >= 0 && semver.Compare(installedVersion, nextMinor) < 0
	}

	nextMajor := incrementMajorVersion(major)
	if nextMajor == "" {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) >= 0 && semver.Compare(installedVersion, nextMajor) < 0
}

// checkTildeRequirement: по semver ~1.2.3 => [1.2.3, 1.3.0) — та же major.minor.
func checkTildeRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixTilde))
	if !semver.IsValid(baseVersion) {
		return false
	}

	majorMinor := semver.MajorMinor(baseVersion)
	if majorMinor == "" {
		return semver.Compare(installedVersion, baseVersion) >= 0
	}

	nextMinor := incrementMinorVersion(majorMinor)
	if nextMinor == "" {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) >= 0 && semver.Compare(installedVersion, nextMinor) < 0
}

func checkGreaterEqualRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixGreaterEqual))
	if !semver.IsValid(baseVersion) {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) >= 0
}

func checkLessEqualRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixLessEqual))
	if !semver.IsValid(baseVersion) {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) <= 0
}

func checkGreaterRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixGreater))
	if !semver.IsValid(baseVersion) {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) > 0
}

func checkLessRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixLess))
	if !semver.IsValid(baseVersion) {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) < 0
}

func checkEqualRequirement(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(strings.TrimPrefix(requirement, requirementPrefixEqual))
	if !semver.IsValid(baseVersion) {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) == 0
}

func checkExactVersion(installedVersion string, requirement string) (compatible bool) {

	baseVersion := normalizeVersion(requirement)
	if !semver.IsValid(baseVersion) {
		return false
	}
	return semver.Compare(installedVersion, baseVersion) == 0
}

func parseMajorVersion(major string) (majorInt int) {

	major = strings.TrimPrefix(major, versionPrefixV)
	if _, err := fmt.Sscanf(major, "%d", &majorInt); err != nil {
		majorInt = 0
		return
	}
	return
}

func incrementMajorVersion(major string) (nextMajor string) {

	majorInt := parseMajorVersion(major)
	return versionPrefixV + fmt.Sprintf("%d", majorInt+1) + ".0.0"
}

func incrementMinorVersion(majorMinor string) (nextMinor string) {

	parts := strings.Split(majorMinor, ".")
	if len(parts) < 2 {
		return ""
	}

	major := parts[0]
	minorStr := strings.TrimPrefix(parts[1], versionPrefixV)
	var minorInt int
	if _, err := fmt.Sscanf(minorStr, "%d", &minorInt); err != nil {
		return ""
	}

	return major + "." + fmt.Sprintf("%d", minorInt+1) + ".0"
}
