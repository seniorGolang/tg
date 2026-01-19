// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"

	"github.com/goccy/go-json"
)

const (
	metadataJSONFilename = "metadata.json"
	metadataFilePerm     = 0600
	sha256FileExt        = ".sha256"
)

// PluginMetadata содержит метаданные установленного плагина.
type PluginMetadata struct {
	RepositoryURL string      `json:"repositoryURL"`
	InstalledAt   time.Time   `json:"installedAt"`
	SourceType    string      `json:"sourceType"` // "github" или "gitlab"
	PluginInfo    plugin.Info `json:"pluginInfo"`
}

func CheckExistingVersion(pluginName string, version string, expectedSHA256 string) (exists bool, checksumMatch bool, err error) {

	pluginsDir := GetPluginsDir()
	versionDir := filepath.Clean(filepath.Join(pluginsDir, pluginName, version))

	var statErr error
	if _, statErr = os.Stat(versionDir); os.IsNotExist(statErr) {
		return false, false, nil
	}

	sha256Path := filepath.Clean(filepath.Join(versionDir, fmt.Sprintf("%s%s", pluginName, sha256FileExt)))
	var sha256Data []byte
	if sha256Data, err = os.ReadFile(sha256Path); err != nil {
		return true, false, nil
	}

	existingSHA256 := strings.TrimSpace(string(sha256Data))
	expectedSHA256 = strings.TrimSpace(expectedSHA256)

	if existingSHA256 != expectedSHA256 {
		return true, false, fmt.Errorf(i18n.Msg("Plugin %s version %s already installed, but checksum mismatch:\n  installed:  %s\n  expected:  %s"), pluginName, version, existingSHA256, expectedSHA256)
	}

	return true, true, nil
}

func createMetadata(repositoryURL string, sourceType string, pluginInfo plugin.Info) (metadata PluginMetadata) {

	metadata = PluginMetadata{
		RepositoryURL: repositoryURL,
		InstalledAt:   time.Now(),
		SourceType:    sourceType,
		PluginInfo:    pluginInfo,
	}
	return
}

// saveMetadata сохраняет метаданные плагина в файл.
func saveMetadata(versionDir string, metadata PluginMetadata) (err error) {

	metadataPath := filepath.Clean(filepath.Join(versionDir, metadataJSONFilename))
	var metadataJSON []byte
	if metadataJSON, err = json.MarshalIndent(metadata, "", "  "); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to serialize metadata: %w"), err)
	}

	if err = os.WriteFile(metadataPath, metadataJSON, metadataFilePerm); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to save metadata.json: %w"), err)
	}

	return
}

// SavePluginMetadata сохраняет только метаданные плагина (файлы уже должны быть на диске).
func SavePluginMetadata(installPath string, pluginInfo plugin.Info, repositoryURL string, sourceType string) (err error) {

	metadata := createMetadata(repositoryURL, sourceType, pluginInfo)
	return saveMetadata(installPath, metadata)
}
