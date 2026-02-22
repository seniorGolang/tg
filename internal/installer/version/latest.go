// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package version

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func Latest(versions []models.Version) (latest models.Version) {

	if len(versions) == 0 {
		return
	}

	latest = versions[0]
	for i := 1; i < len(versions); i++ {
		if Compare(versions[i], latest) > 0 {
			latest = versions[i]
		}
	}

	return
}
