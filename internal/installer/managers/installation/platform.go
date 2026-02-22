// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"fmt"
	"runtime"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func (m *manager) selectURLForDownload(downloads []models.PlatformDownload) (url string, err error) {

	currentOS := runtime.GOOS
	currentArch := runtime.GOARCH

	var bestMatch *models.PlatformDownload
	bestPriority := 0

	for i := range downloads {
		download := &downloads[i]
		priority := m.calculatePriority(download, currentOS, currentArch)
		if priority > bestPriority {
			bestPriority = priority
			bestMatch = download
		}
	}

	if bestMatch == nil {
		return "", fmt.Errorf(i18n.Msg("No suitable URL found for platform %s/%s"), currentOS, currentArch)
	}

	return bestMatch.URL, nil
}

// calculatePriority: 4 = OS+Arch, 3 = только OS, 2 = только Arch, 1 = оба пустые, 0 = не подходит.
func (m *manager) calculatePriority(download *models.PlatformDownload, currentOS string, currentArch string) (priority int) {

	if download.OS == currentOS && download.Arch == currentArch {
		return 4
	}

	if download.OS == currentOS && download.Arch == "" {
		return 3
	}

	if download.OS == "" && download.Arch == currentArch {
		return 2
	}

	if download.OS == "" && download.Arch == "" {
		return 1
	}

	return 0
}
