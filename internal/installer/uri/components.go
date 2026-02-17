// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func (u *URI) Source() (source string) {

	if u == nil {
		return ""
	}
	return u.source
}

func (u *URI) Package() (packageName string) {

	if u == nil {
		return ""
	}
	return u.packageName
}

func (u *URI) Version() (version models.Version) {

	if u == nil {
		return models.Version{}
	}
	return u.version
}
