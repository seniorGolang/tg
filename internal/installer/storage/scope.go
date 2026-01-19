// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultScopeName имя scope по умолчанию.
	DefaultScopeName = "default"
)

// GlobalConfig представляет глобальную конфигурацию.
type GlobalConfig struct {
	CurrentScope string `yaml:"current_scope"`
}

// ScopeConfig представляет конфигурацию scope.
type ScopeConfig struct {
	Name          string `yaml:"name"`
	InstallPrefix string `yaml:"install_prefix"`
	BinDir        string `yaml:"bin_dir"`
	LibDir        string `yaml:"lib_dir"`
	ConfigDir     string `yaml:"config_dir"`
}

func GetCurrentScope() (scopeName string, err error) {

	return GetEffectiveScope("")
}

// GetEffectiveScope: приоритет scopeFromFlag, иначе config.yml, иначе "default".
func GetEffectiveScope(scopeFromFlag string) (scopeName string, err error) {

	if scopeFromFlag != "" {
		scopeName = scopeFromFlag
		return
	}

	configFile := GetGlobalConfigFile()
	var statErr error
	if _, statErr = os.Stat(configFile); os.IsNotExist(statErr) {
		scopeName = DefaultScopeName
		return
	}

	var data []byte
	if data, err = os.ReadFile(configFile); err != nil {
		scopeName = DefaultScopeName
		return
	}

	var config GlobalConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		scopeName = DefaultScopeName
		return
	}

	if config.CurrentScope == "" {
		scopeName = DefaultScopeName
		return
	}

	scopeName = config.CurrentScope
	return
}

func SetCurrentScope(scopeName string) (err error) {

	configFile := GetGlobalConfigFile()
	if err = EnsureDir(filepath.Dir(configFile)); err != nil {
		return
	}

	config := GlobalConfig{
		CurrentScope: scopeName,
	}

	var data []byte
	if data, err = yaml.Marshal(&config); err != nil {
		return
	}

	err = os.WriteFile(configFile, data, FilePermFile)
	return
}

// LoadScopeConfig загружает конфигурацию scope.
func LoadScopeConfig(scopeName string) (config *ScopeConfig, err error) {

	configFile := GetConfigFile(scopeName)
	var statErr error
	if _, statErr = os.Stat(configFile); os.IsNotExist(statErr) {
		config = getDefaultScopeConfig(scopeName)
		return
	}

	var data []byte
	if data, err = os.ReadFile(configFile); err != nil {
		config = nil
		return
	}

	config = &ScopeConfig{}
	if err = yaml.Unmarshal(data, config); err != nil {
		config = nil
		return
	}

	return
}

// SaveScopeConfig сохраняет конфигурацию scope.
func SaveScopeConfig(scopeName string, config *ScopeConfig) (err error) {

	configFile := GetConfigFile(scopeName)
	if err = EnsureDir(filepath.Dir(configFile)); err != nil {
		return
	}

	var data []byte
	if data, err = yaml.Marshal(config); err != nil {
		return
	}

	err = os.WriteFile(configFile, data, FilePermFile)
	return
}

func getDefaultScopeConfig(scopeName string) (config *ScopeConfig) {

	home := GetHomeDir()
	scopeDir := filepath.Join(home, ScopesDirName, scopeName)
	return &ScopeConfig{
		Name:          scopeName,
		InstallPrefix: scopeDir,
		BinDir:        filepath.Join(scopeDir, BinDirName),
		LibDir:        filepath.Join(scopeDir, LibDirName),
		ConfigDir:     filepath.Join(scopeDir, EtcDirName),
	}
}
