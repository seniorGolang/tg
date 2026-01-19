// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

// ParseVersion парсит версию URI в структуру Version.
func (u *URI) ParseVersion() (v models.Version, err error) {

	if u.version.Original == "" {
		return
	}

	v, err = version.Parse(u.version.Original)
	return
}

// CompareVersions сравнивает версию URI с другой версией.
func (u *URI) CompareVersions(other models.Version) (result int, err error) {

	var v models.Version
	if v, err = u.ParseVersion(); err != nil {
		return
	}

	if v.Original == "" {
		return 0, nil
	}

	result = version.Compare(v, other)
	return
}

func (u *URI) MatchVersion(constraint string) (matches bool, err error) {

	var v models.Version
	if v, err = u.ParseVersion(); err != nil {
		return
	}

	if v.Original == "" {
		matches = constraint == "" || constraint == version.LatestVersion
		return
	}

	matches = version.Match(constraint, v)
	return
}

func (u *URI) IsPreRelease() (isPreRelease bool, err error) {

	var v models.Version
	if v, err = u.ParseVersion(); err != nil {
		return
	}

	if v.Original == "" {
		return false, nil
	}

	isPreRelease = version.IsPreRelease(v)
	return
}

func (u *URI) HasVersionConstraint() (hasConstraint bool) {

	return u.hasVersionConstraint()
}

// FilterVersionsByConstraint фильтрует список версий по ограничению версии URI.
func (u *URI) FilterVersionsByConstraint(versions []models.Version) (filtered []models.Version) {

	return version.FilterByConstraint(u.version.Original, versions)
}

// FindMatchingVersion находит версию из списка, соответствующую ограничению версии URI.
func (u *URI) FindMatchingVersion(versions []models.Version) (matched models.Version) {

	return version.FindMatching(u.version.Original, versions)
}
