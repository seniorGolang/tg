// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

import (
	"time"
)

// Installation представляет запись об установленном пакете.
type Installation struct {
	ID           string          `yaml:"id"`
	Source       string          `yaml:"source,omitempty"`
	Package      string          `yaml:"package"`
	Version      string          `yaml:"version"`
	Descr        string          `yaml:"descr,omitempty"`
	InstalledAt  time.Time       `yaml:"installed_at"`
	Files        []InstalledFile `yaml:"files"`
	Dependencies []string        `yaml:"dependencies,omitempty"`

	// Поля плагина (заполняются только если пакет является плагином)
	Commands         []CommandInfo     `yaml:"commands,omitempty"`
	Options          []OptionInfo      `yaml:"options,omitempty"`
	Kind             string            `yaml:"kind,omitempty"`
	Silent           bool              `yaml:"silent,omitempty"`
	Always           bool              `yaml:"always,omitempty"`
	AllowedPaths     map[string]string `yaml:"allowed_paths,omitempty"`
	AllowedEnvVars   []string          `yaml:"allowed_env_vars,omitempty"`
	AllowedHosts     []string          `yaml:"allowed_hosts,omitempty"`
	AllowedShellCMDs []string          `yaml:"allowed_shell_cmds,omitempty"`
	InitPkgs         []string          `yaml:"init_pkgs,omitempty"`
}

// InstalledFile описывает установленный файл.
type InstalledFile struct {
	Path     string `yaml:"path"`
	Source   string `yaml:"source,omitempty"`
	Checksum string `yaml:"checksum,omitempty"`
	Size     int64  `yaml:"size"`
}

// CommandInfo описывает команду плагина для регистрации.
type CommandInfo struct {
	Path        []string     `yaml:"path"`
	Description string       `yaml:"description"`
	Options     []OptionInfo `yaml:"options,omitempty"`
}

// OptionInfo описывает опцию плагина.
type OptionInfo struct {
	Name         string `yaml:"name"`
	Short        string `yaml:"short,omitempty"`
	Type         string `yaml:"type"`
	Description  string `yaml:"description"`
	Required     bool   `yaml:"required,omitempty"`
	Default      any    `yaml:"default,omitempty"`
	IsPositional bool   `yaml:"is_positional,omitempty"`
}

// InstallationDatabase представляет базу данных установок.
type InstallationDatabase struct {
	Version   string         `yaml:"version"`
	Installed []Installation `yaml:"installed"`
	UpdatedAt time.Time      `yaml:"updated_at"`
}
