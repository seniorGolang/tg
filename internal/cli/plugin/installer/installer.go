// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installer

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/installer/github"
	"github.com/seniorGolang/tg/v3/internal/cli/plugin/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/cli/plugin/installer/validator"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"

	"github.com/goccy/go-json"
)

// PluginInstaller представляет установщик плагинов.
type PluginInstaller struct {
	source  PluginSource
	repoURL string
}

func NewPluginInstaller(repoURL string) (installer *PluginInstaller, err error) {

	var githubClient *github.Client
	if githubClient, err = github.NewClient(repoURL); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "GitHub client", err)
	}

	installer = &PluginInstaller{
		source:  githubClient,
		repoURL: repoURL,
	}
	return
}

// InstallPlugin: installPath — путь относительно projectRoot (например, "plugins/test/1.2.10").
// logger - логгер для вывода информации о процессе установки.
// Возвращает путь к директории установки (относительный) и размеры скачанных файлов.
func (i *PluginInstaller) InstallPlugin(ctx context.Context, pluginName string, version string, installPath string, logger plugin.Logger) (installedPath string, jsonSize int64, tgpSize int64, err error) {

	var resolvedPluginName, resolvedVersion string
	if resolvedPluginName, resolvedVersion, err = i.resolveVersion(ctx, pluginName, version); err != nil {
		return
	}

	pluginName = resolvedPluginName
	version = resolvedVersion

	var jsonPath, tgpPath, sha256Path string
	if jsonPath, jsonSize, tgpPath, tgpSize, sha256Path, _, err = i.source.DownloadPluginFiles(ctx, resolvedPluginName, resolvedVersion, installPath); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Failed to download plugin files %s version %s: %w"), resolvedPluginName, resolvedVersion, err)
	}

	var pluginJSON []byte
	if pluginJSON, err = os.ReadFile(jsonPath); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Failed to read plugin.json: %w"), err)
	}

	if len(pluginJSON) == 0 {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("plugin.json is empty: %s"), jsonPath)
	}

	var manifest PluginManifest
	if err = json.Unmarshal(pluginJSON, &manifest); err != nil {
		logger.Debug(i18n.Msg("Error parsing plugin.json"), "content", string(pluginJSON), "error", err)
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Failed to parse plugin.json (size: %d bytes): %w"), len(pluginJSON), err)
	}

	pluginInfo := plugin.Info{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: manifest.Description,
		Author:      manifest.Author,
		License:     manifest.License,
		Commands:    []plugin.Command{},
	}

	if pluginName == "" {
		pluginName = pluginInfo.Name
	}

	var pluginSHA256 []byte
	if pluginSHA256, err = os.ReadFile(sha256Path); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Failed to read %s.sha256: %w"), pluginName, err)
	}

	expectedSHA256 := strings.TrimSpace(string(pluginSHA256))
	var exists, checksumMatch bool
	if exists, checksumMatch, err = storage.CheckExistingVersion(pluginName, version, expectedSHA256); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, err
	}

	if exists && checksumMatch {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Plugin %s version %s already installed with same checksum"), pluginName, version)
	}

	var pluginTGP []byte
	if pluginTGP, err = os.ReadFile(tgpPath); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Failed to read %s.tgp: %w"), pluginName, err)
	}

	logger.Info(i18n.Msg("Validating SHA256 checksum"))
	if err = validator.ValidateChecksum(pluginTGP, pluginSHA256); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Checksum validation failed: %w"), err)
	}

	var wasmBytes []byte
	if wasmBytes, err = plugin.DecodeTGPBytes(pluginTGP); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, err
	}

	logger.Info(i18n.Msg("Validating metadata"))
	if err = validator.ValidateMetadata(pluginInfo, pluginName, version); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Metadata validation failed: %w"), err)
	}

	logger.Info(i18n.Msg("Validating WASM structure"))
	if err = validator.ValidateWASM(wasmBytes, pluginInfo); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("WASM structure validation failed: %w"), err)
	}

	logger.Info(i18n.Msg("Saving plugin"))
	if err = storage.SavePluginMetadata(installPath, pluginInfo, i.repoURL, "github"); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, 0, fmt.Errorf(i18n.Msg("Failed to save plugin metadata: %w"), err)
	}

	installedPath = installPath
	return
}

func (i *PluginInstaller) ListAvailablePlugins(ctx context.Context) (plugins []PluginInfo, err error) {

	return i.source.ListPlugins(ctx)
}

func (i *PluginInstaller) ListAvailableVersions(ctx context.Context, pluginName string) (versions []VersionInfo, err error) {

	return i.source.ListVersions(ctx, pluginName)
}
