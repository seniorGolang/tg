// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package version

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func FilterByConstraint(constraint string, versions []models.Version) (filtered []models.Version) {

	if constraint == "" {
		return versions
	}

	filtered = make([]models.Version, 0, len(versions))
	for _, v := range versions {
		if Match(constraint, v) {
			filtered = append(filtered, v)
		}
	}

	return
}

// FindMatching находит версию из списка, соответствующую ограничению.
// Если найдено несколько версий, возвращает последнюю.
func FindMatching(constraint string, versions []models.Version) (matched models.Version) {

	filtered := FilterByConstraint(constraint, versions)
	if len(filtered) == 0 {
		return
	}

	return Latest(filtered)
}
