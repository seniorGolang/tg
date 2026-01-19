// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package database

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/storage/yamlstore"
)

// manager реализует DatabaseManager.
type manager struct {
	scopeName string
	store     *yamlstore.YAMLCollectionStore[models.Installation, string]
}

func NewManager(scopeName string) (mgr managers.DatabaseManager) {
	dbFile := storage.GetPackagesDBFile(scopeName)
	getID := func(inst models.Installation) string {
		return inst.ID
	}

	mgr = &manager{
		scopeName: scopeName,
		store:     yamlstore.NewYAMLCollectionStore(dbFile, getID),
	}
	return
}

// RecordInstallation: уникальность по ID (Package+Version). При существующем ID — обновление (включая source).
func (m *manager) RecordInstallation(ctx context.Context, installation *models.Installation) (err error) {
	if installation == nil {
		err = errors.New(i18n.Msg("installation cannot be nil"))
		return
	}

	dbFile := storage.GetPackagesDBFile(m.scopeName)
	if err = storage.EnsureDir(filepath.Dir(dbFile)); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
		return
	}

	if err = m.store.Add(*installation); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
		return
	}

	if err = m.store.Save(); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to serialize database: %w"), err)
		return
	}

	return
}

func (m *manager) RemoveInstallation(ctx context.Context, packageID string) (err error) {
	if err = m.store.Remove(packageID); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
		return
	}

	if err = m.store.Save(); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to serialize database: %w"), err)
		return
	}

	return
}

func (m *manager) GetInstallation(ctx context.Context, packageID string) (installation *models.Installation, err error) {
	var inst models.Installation
	var found bool
	if inst, found, err = m.store.FindByID(packageID); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
		return
	}

	if !found {
		err = errors.New(i18n.Msg("Installation not found"))
		return
	}

	installation = &inst
	return
}

func (m *manager) ListInstallations(ctx context.Context) (installations []models.Installation, err error) {
	if installations, err = m.store.GetAll(); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
		return
	}

	return
}

// FindByPackage ищет установку по источнику и имени пакета.
// Если source пустой, ищет только по имени пакета (игнорирует source).
func (m *manager) FindByPackage(ctx context.Context, source string, packageName string) (installation *models.Installation, err error) {
	var installations []models.Installation
	installations, err = m.ListInstallations(ctx)
	if err != nil {
		return
	}

	for _, inst := range installations {
		if inst.Package == packageName {
			// Если source пустой, возвращаем первую найденную установку с таким именем пакета
			if source == "" {
				installation = &inst
				return
			}
			// Если source указан, проверяем точное совпадение
			if inst.Source == source {
				installation = &inst
				return
			}
		}
	}

	err = errors.New(i18n.Msg("Installation not found"))
	return
}
