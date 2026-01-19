// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package scope

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/managers/manifest"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

const (
	scopesDirName = "scopes"
	binDirName    = "bin"
	libDirName    = "lib"
	etcDirName    = "etc"
)

// manager реализует ScopeManager.
type manager struct{}

func NewManager() (mgr managers.ScopeManager) {

	return &manager{}
}

func (m *manager) CreateScope(ctx context.Context, name string, options *models.ScopeOptions) (err error) {

	scopeDir := storage.GetScopeDir(name)
	var statErr error
	if _, statErr = os.Stat(scopeDir); statErr == nil {
		err = fmt.Errorf(i18n.Msg("Scope %s already exists"), name)
		return
	}

	if err = storage.EnsureDir(scopeDir); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "scope directory", err)
		return
	}

	var config *models.ScopeConfig
	if options != nil && options.From != "" {
		var sourceConfig *storage.ScopeConfig
		if sourceConfig, err = storage.LoadScopeConfig(options.From); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to load configuration from scope %s: %w"), options.From, err)
			return
		}
		config = &models.ScopeConfig{
			Name:          name,
			InstallPrefix: sourceConfig.InstallPrefix,
			BinDir:        sourceConfig.BinDir,
			LibDir:        sourceConfig.LibDir,
			ConfigDir:     sourceConfig.ConfigDir,
		}
	} else {
		home := storage.GetHomeDir()
		config = &models.ScopeConfig{
			Name:          name,
			InstallPrefix: filepath.Join(home, scopesDirName, name),
			BinDir:        filepath.Join(home, scopesDirName, name, binDirName),
			LibDir:        filepath.Join(home, scopesDirName, name, libDirName),
			ConfigDir:     filepath.Join(home, scopesDirName, name, etcDirName),
		}
	}

	scopeConfig := &storage.ScopeConfig{
		Name:          config.Name,
		InstallPrefix: config.InstallPrefix,
		BinDir:        config.BinDir,
		LibDir:        config.LibDir,
		ConfigDir:     config.ConfigDir,
	}

	if err = storage.SaveScopeConfig(name, scopeConfig); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to save scope configuration: %w"), err)
		return
	}

	if err = storage.EnsureDir(storage.GetCatalogDir(name)); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "catalog directory", err)
		return
	}

	if err = storage.EnsureDir(storage.GetInstalledDir(name)); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "installations directory", err)
		return
	}

	return
}

// UseScope переключается на указанный scope.
func (m *manager) UseScope(ctx context.Context, name string) (err error) {

	scopeDir := storage.GetScopeDir(name)
	var statErr error
	if _, statErr = os.Stat(scopeDir); os.IsNotExist(statErr) {
		err = fmt.Errorf(i18n.Msg("Scope %s does not exist"), name)
		return
	}

	if err = storage.SetCurrentScope(name); err != nil {
		return
	}
	return
}

// DeleteScope удаляет scope.
func (m *manager) DeleteScope(ctx context.Context, name string, force bool) (err error) {

	if name == storage.DefaultScopeName && !force {
		err = errors.New(i18n.Msg("Cannot delete scope default without --force flag"))
		return
	}

	scopeDir := storage.GetScopeDir(name)
	var statErr error
	if _, statErr = os.Stat(scopeDir); os.IsNotExist(statErr) {
		err = fmt.Errorf(i18n.Msg("Scope %s does not exist"), name)
		return
	}

	if !force {
		installedDir := storage.GetInstalledDir(name)
		var entries []os.DirEntry
		if entries, err = os.ReadDir(installedDir); err == nil && len(entries) > 0 {
			err = fmt.Errorf(i18n.Msg("Scope %s has installed packages, use --force for forced deletion"), name)
			return
		}
	}

	if err = os.RemoveAll(scopeDir); err != nil {
		return
	}
	return
}

func (m *manager) ListScopes(ctx context.Context) (scopes []models.ScopeInfo, err error) {

	home := storage.GetHomeDir()
	scopesDir := filepath.Join(home, storage.ScopesDirName)

	var currentScope string
	currentScope, _ = storage.GetCurrentScope()

	scopes = make([]models.ScopeInfo, 0)
	var hasDefault bool

	var entries []os.DirEntry
	if _, statErr := os.Stat(scopesDir); statErr == nil {
		if entries, err = os.ReadDir(scopesDir); err != nil {
			scopes = nil
			return
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			scopeName := entry.Name()
			if scopeName == storage.DefaultScopeName {
				hasDefault = true
			}

			var scopeInfo models.ScopeInfo
			if scopeInfo, err = m.getScopeInfo(ctx, scopeName, currentScope); err != nil {
				scopes = nil
				return
			}

			scopes = append(scopes, scopeInfo)
		}
	}

	if !hasDefault {
		var defaultScopeInfo models.ScopeInfo
		if defaultScopeInfo, err = m.getScopeInfo(ctx, storage.DefaultScopeName, currentScope); err != nil {
			scopes = nil
			return
		}

		scopes = append(scopes, defaultScopeInfo)
	}

	return
}

func (m *manager) getScopeInfo(ctx context.Context, scopeName string, currentScope string) (scopeInfo models.ScopeInfo, err error) {

	manifestMgr := manifest.NewManager(scopeName)

	var allManifests []managers.ManifestWithSource
	if allManifests, err = manifestMgr.GetAllManifests(ctx); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get manifests for scope %s: %w"), scopeName, err)
		return
	}

	manifestCount := len(allManifests)

	home := storage.GetHomeDir()
	scopesDir := filepath.Join(home, storage.ScopesDirName)
	scopeDir := filepath.Join(scopesDir, scopeName)

	var packageCount int
	installedDir := filepath.Join(scopeDir, storage.InstalledDirName)
	var installedEntries []os.DirEntry
	if installedEntries, err = os.ReadDir(installedDir); err == nil {
		for _, e := range installedEntries {
			if e.IsDir() {
				packageCount++
			}
		}
	}

	scopeInfo = models.ScopeInfo{
		Name:          scopeName,
		IsActive:      scopeName == currentScope,
		PackageCount:  packageCount,
		ManifestCount: manifestCount,
	}

	return
}

func (m *manager) GetCurrentScope(ctx context.Context) (name string, err error) {

	name, err = storage.GetCurrentScope()
	return
}

func (m *manager) GetScopeConfig(ctx context.Context, scopeName string) (config *models.ScopeConfig, err error) {

	var scopeConfig *storage.ScopeConfig
	if scopeConfig, err = storage.LoadScopeConfig(scopeName); err != nil {
		config = nil
		err = fmt.Errorf(i18n.Msg("Failed to load scope configuration: %w"), err)
		return
	}

	config = &models.ScopeConfig{
		Name:          scopeConfig.Name,
		InstallPrefix: scopeConfig.InstallPrefix,
		BinDir:        scopeConfig.BinDir,
		LibDir:        scopeConfig.LibDir,
		ConfigDir:     scopeConfig.ConfigDir,
	}
	return
}
