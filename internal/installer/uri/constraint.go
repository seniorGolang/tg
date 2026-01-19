// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"strings"

	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

func (u *URI) hasVersionConstraint() (hasConstraint bool) {

	if u.version.Original == "" {
		return false
	}

	if u.version.Original == version.LatestVersion {
		return true
	}

	return strings.HasPrefix(u.version.Original, ">=") ||
		strings.HasPrefix(u.version.Original, "<=") ||
		strings.HasPrefix(u.version.Original, ">") ||
		strings.HasPrefix(u.version.Original, "<") ||
		strings.HasPrefix(u.version.Original, "~") ||
		strings.HasPrefix(u.version.Original, "^")
}
