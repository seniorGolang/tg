// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"os"
	"path/filepath"
)

func GetHomeDir() (home string) {

	home = os.Getenv("TG_HOME")
	if home != "" {
		return
	}

	var err error
	var userHome string
	if userHome, err = os.UserHomeDir(); err != nil {
		home = DefaultHomeDir
		return
	}

	return filepath.Join(userHome, HomeDirName)
}

func GetScopeDir(scopeName string) (dir string) {

	return filepath.Join(GetHomeDir(), ScopesDirName, scopeName)
}

func GetCatalogDir(scopeName string) (dir string) {

	return filepath.Join(GetScopeDir(scopeName), CatalogDirName)
}

func GetManifestDir(scopeName string, normalizedURL string) (dir string) {

	return filepath.Join(GetCatalogDir(scopeName), normalizedURL)
}

func GetInstalledDir(scopeName string) (dir string) {

	return filepath.Join(GetScopeDir(scopeName), InstalledDirName)
}

func GetGlobalConfigFile() (path string) {

	return filepath.Join(GetHomeDir(), ConfigFileName)
}

func GetPackagesDBFile(scopeName string) (path string) {

	return filepath.Join(GetInstalledDir(scopeName), PackagesDBFileName)
}

// GetCacheDir: если конфиг scope недоступен, используется scopeDir/cache.
func GetCacheDir(scopeName string) (dir string) {

	var err error
	var scopeConfig *ScopeConfig
	if scopeConfig, err = LoadScopeConfig(scopeName); err != nil {
		scopeDir := GetScopeDir(scopeName)
		return filepath.Join(scopeDir, CacheDirName)
	}
	return filepath.Join(scopeConfig.InstallPrefix, CacheDirName)
}
