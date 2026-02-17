// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

func (u *URI) ParseVersion() (v models.Version, err error) {

	if u.version.Original == "" {
		return
	}

	return version.Parse(u.version.Original)
}

func (u *URI) CompareVersions(other models.Version) (result int, err error) {

	var v models.Version
	if v, err = u.ParseVersion(); err != nil {
		return
	}

	if v.Original == "" {
		return 0, nil
	}

	return version.Compare(v, other), nil
}

func (u *URI) MatchVersion(constraint string) (matches bool, err error) {

	var v models.Version
	if v, err = u.ParseVersion(); err != nil {
		return
	}

	if v.Original == "" {
		return constraint == "" || constraint == version.LatestVersion, nil
	}

	return version.Match(constraint, v), nil
}

func (u *URI) IsPreRelease() (isPreRelease bool, err error) {

	var v models.Version
	if v, err = u.ParseVersion(); err != nil {
		return
	}

	if v.Original == "" {
		return false, nil
	}

	return version.IsPreRelease(v), nil
}
