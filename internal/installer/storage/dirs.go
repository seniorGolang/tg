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

	var userHome string
	var err error
	if userHome, err = os.UserHomeDir(); err != nil {
		home = DefaultHomeDir
		return
	}

	home = filepath.Join(userHome, HomeDirName)
	return
}

func GetScopeDir(scopeName string) (dir string) {

	home := GetHomeDir()
	dir = filepath.Join(home, ScopesDirName, scopeName)
	return
}

func GetCatalogDir(scopeName string) (dir string) {

	scopeDir := GetScopeDir(scopeName)
	dir = filepath.Join(scopeDir, CatalogDirName)
	return
}

func GetManifestDir(scopeName string, normalizedURL string) (dir string) {

	catalogDir := GetCatalogDir(scopeName)
	dir = filepath.Join(catalogDir, normalizedURL)
	return
}

func GetInstalledDir(scopeName string) (dir string) {

	scopeDir := GetScopeDir(scopeName)
	dir = filepath.Join(scopeDir, InstalledDirName)
	return
}

func GetPackageDir(scopeName string, packageID string) (dir string) {

	installedDir := GetInstalledDir(scopeName)
	dir = filepath.Join(installedDir, packageID)
	return
}

func GetGlobalConfigFile() (path string) {

	home := GetHomeDir()
	path = filepath.Join(home, ConfigFileName)
	return
}

func GetPackagesDBFile(scopeName string) (path string) {

	installedDir := GetInstalledDir(scopeName)
	path = filepath.Join(installedDir, PackagesDBFileName)
	return
}

// GetCacheDir: если конфиг scope недоступен, используется scopeDir/cache.
func GetCacheDir(scopeName string) (dir string) {

	var scopeConfig *ScopeConfig
	var err error
	scopeConfig, err = LoadScopeConfig(scopeName)
	if err != nil {
		scopeDir := GetScopeDir(scopeName)
		return filepath.Join(scopeDir, CacheDirName)
	}
	return filepath.Join(scopeConfig.InstallPrefix, CacheDirName)
}
