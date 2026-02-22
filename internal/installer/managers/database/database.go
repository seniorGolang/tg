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

type manager struct {
	scopeName string
	store     *yamlstore.YAMLCollectionStore[models.Installation, string]
}

func NewManager(scopeName string) (mgr managers.DatabaseManager) {

	dbFile := storage.GetPackagesDBFile(scopeName)
	getID := func(inst models.Installation) string {
		return inst.ID
	}

	return &manager{
		scopeName: scopeName,
		store:     yamlstore.NewYAMLCollectionStore(dbFile, getID),
	}
}

// RecordInstallation: уникальность по ID (Package+Version). При существующем ID — обновление (включая source).
func (m *manager) RecordInstallation(ctx context.Context, installation *models.Installation) (err error) {

	if installation == nil {
		return errors.New(i18n.Msg("installation cannot be nil"))
	}

	dbFile := storage.GetPackagesDBFile(m.scopeName)
	if err = storage.EnsureDir(filepath.Dir(dbFile)); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
	}

	if err = m.store.Add(*installation); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
	}

	if err = m.store.Save(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to serialize database: %w"), err)
	}

	return
}

func (m *manager) RemoveInstallation(ctx context.Context, packageID string) (err error) {

	if err = m.store.Remove(packageID); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
	}

	if err = m.store.Save(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to serialize database: %w"), err)
	}

	return
}

func (m *manager) GetInstallation(ctx context.Context, packageID string) (installation *models.Installation, err error) {

	var inst models.Installation
	var found bool
	if inst, found, err = m.store.FindByID(packageID); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
	}

	if !found {
		return nil, errors.New(i18n.Msg("Installation not found"))
	}

	return &inst, nil
}

func (m *manager) ListInstallations(ctx context.Context) (installations []models.Installation, err error) {

	if installations, err = m.store.GetAll(); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to read database: %w"), err)
	}

	return
}

// FindByPackage ищет установку по источнику и имени пакета.
// Если source пустой, ищет только по имени пакета (игнорирует source).
func (m *manager) FindByPackage(ctx context.Context, source string, packageName string) (installation *models.Installation, err error) {

	var installations []models.Installation
	if installations, err = m.ListInstallations(ctx); err != nil {
		return
	}

	for _, inst := range installations {
		if inst.Package == packageName {
			if source == "" {
				return &inst, nil
			}
			if inst.Source == source {
				return &inst, nil
			}
		}
	}

	return nil, errors.New(i18n.Msg("Installation not found"))
}
