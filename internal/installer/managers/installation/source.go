// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// findPackageSource находит source пакета из каталога манифестов.
func (m *manager) findPackageSource(ctx context.Context, pkg *models.Package) (source string, err error) {

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	var allManifests []managers.ManifestWithSource
	if allManifests, err = m.manifestManager.GetAllManifests(ctx); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get manifests: %w"), err)
		return
	}

	for _, manifestWithSource := range allManifests {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}
		for _, manifestPkg := range manifestWithSource.Manifest.Packages {
			if manifestPkg.Name == pkg.Name {
				source = manifestWithSource.Source
				return
			}
		}
	}

	err = errors.New(i18n.Msg("Package source not found"))
	return
}
