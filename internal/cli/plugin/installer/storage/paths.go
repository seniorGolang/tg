// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	tgDirName      = "tg"
	configDirName  = ".config"
	pluginsDirName = "plugins"
)

func GetPluginsDir() (pluginsDir string) {

	var homeDir string
	var err error
	if homeDir, err = os.UserHomeDir(); err != nil {
		return ""
	}

	pluginsPath := filepath.Join(homeDir, configDirName, tgDirName, pluginsDirName)
	pluginsPath = filepath.Clean(pluginsPath)

	if !filepath.IsAbs(pluginsPath) {
		var absPath string
		if absPath, err = filepath.Abs(pluginsPath); err == nil {
			return filepath.Clean(absPath)
		}
	}

	return pluginsPath
}

// GetPluginInstallPath формирует путь установки плагина с учетом источника (как в Go).
// Формат: {pluginsDir}/{source}/{owner}/{repo}/{plugin-name}/{version}
// Для GitHub: github.com/owner/repo/plugin-name/version
func GetPluginInstallPath(repoURL string, pluginName string, version string) (installPath string, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(repoURL); err != nil {
		installPath = ""
		return
	}

	sourcePath := filepath.Join(parsedURL.Host, strings.TrimPrefix(parsedURL.Path, "/"))
	pluginsDir := GetPluginsDir()
	installPath = filepath.Join(pluginsDir, sourcePath, pluginName, version)

	return
}

// GetPluginInstallPathRelative: формат {source}/{owner}/{repo}/{plugin-name}/{version}.
func GetPluginInstallPathRelative(repoURL string, pluginName string, version string) (installPath string, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(repoURL); err != nil {
		installPath = ""
		return
	}

	sourcePath := filepath.Join(parsedURL.Host, strings.TrimPrefix(parsedURL.Path, "/"))
	installPath = filepath.Join(sourcePath, pluginName, version)

	return
}

// GetAbsoluteInstallPath: если installPath абсолютный — как есть, иначе объединяет с GetPluginsDir().
func GetAbsoluteInstallPath(installPath string) (absolutePath string, err error) {

	if filepath.IsAbs(installPath) {
		absolutePath = filepath.Clean(installPath)
		return
	}

	pluginsDir := GetPluginsDir()
	if pluginsDir == "" {
		absolutePath = ""
		err = errors.New(i18n.Msg("Failed to get plugins directory"))
		return
	}

	absolutePath = filepath.Join(pluginsDir, installPath)
	return filepath.Clean(absolutePath), nil
}

// CleanupInstallPath удаляет директорию установки плагина и все её содержимое.
// installPath может быть как абсолютным, так и относительным.
func CleanupInstallPath(installPath string) (err error) {

	var absolutePath string
	if absolutePath, err = GetAbsoluteInstallPath(installPath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get absolute path for cleanup: %w"), err)
	}

	var statErr error
	if _, statErr = os.Stat(absolutePath); os.IsNotExist(statErr) {
		return nil
	}

	if err = os.RemoveAll(absolutePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to remove directory %s: %w"), absolutePath, err)
	}

	return
}
