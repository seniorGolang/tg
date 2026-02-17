// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultScopeName = "default"
)

type GlobalConfig struct {
	CurrentScope string `yaml:"current_scope"`
}

type ScopeConfig struct {
	Name          string `yaml:"name"`
	BinDir        string `yaml:"bin_dir"`
	LibDir        string `yaml:"lib_dir"`
	ConfigDir     string `yaml:"config_dir"`
	InstallPrefix string `yaml:"install_prefix"`
}

// GetEffectiveScope: приоритет name[0] (если передан), иначе config.yml, иначе "default".
func GetEffectiveScope(name ...string) (scopeName string, err error) {

	if len(name) > 0 && name[0] != "" {
		scopeName = name[0]
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

	if err = os.WriteFile(configFile, data, FilePermFile); err != nil {
		return
	}

	return
}

func LoadScopeConfig(scopeName string) (config *ScopeConfig, err error) {

	home := GetHomeDir()
	scopeDir := filepath.Join(home, ScopesDirName, scopeName)
	return &ScopeConfig{
		Name:          scopeName,
		InstallPrefix: scopeDir,
		BinDir:        filepath.Join(scopeDir, BinDirName),
		LibDir:        filepath.Join(scopeDir, LibDirName),
		ConfigDir:     filepath.Join(scopeDir, EtcDirName),
	}, nil
}
