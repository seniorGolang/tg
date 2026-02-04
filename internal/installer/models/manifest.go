// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

// Manifest описывает конкретную версию релиза с пакетами и/или ссылками на другие манифесты.
type Manifest struct {
	Version   string        `yaml:"version"`
	Manifests []ManifestRef `yaml:"manifests,omitempty"`
	Packages  []Package     `yaml:"packages,omitempty"`
}

// ManifestRef представляет ссылку на другой манифест.
type ManifestRef struct {
	URL string `yaml:"url"`
}

// Package описывает пакет из релиза.
type Package struct {
	Name         string             `yaml:"name"`
	Descr        string             `yaml:"descr,omitempty"`
	Hidden       bool               `yaml:"hidden,omitempty"`
	Alias        string             `yaml:"alias,omitempty"`
	Downloads    []PlatformDownload `yaml:"downloads"`
	Files        []FileInstallation `yaml:"files"`
	Scripts      *Scripts           `yaml:"scripts,omitempty"`
	Dependencies []string           `yaml:"dependencies,omitempty"`
}

// PlatformDownload содержит информацию о загрузке.
type PlatformDownload struct {
	OS   string `yaml:"os,omitempty"`
	Arch string `yaml:"arch,omitempty"`
	URL  string `yaml:"url"`
}

// FileInstallation описывает установку файла.
type FileInstallation struct {
	File        string `yaml:"file,omitempty"`
	Source      string `yaml:"source,omitempty"`
	Destination string `yaml:"destination"`
	Checksum    string `yaml:"checksum,omitempty"`
}

// Scripts содержит скрипты для выполнения.
type Scripts struct {
	PreInstall    *ScriptAction `yaml:"pre_install,omitempty"`
	PostInstall   *ScriptAction `yaml:"post_install,omitempty"`
	PreUninstall  *ScriptAction `yaml:"pre_uninstall,omitempty"`
	PostUninstall *ScriptAction `yaml:"post_uninstall,omitempty"`
}

// ScriptAction описывает действие скрипта.
type ScriptAction struct {
	Script string `yaml:"script,omitempty"`
	Source string `yaml:"source,omitempty"`
	Exec   string `yaml:"exec"`
}

// Dependency описывает зависимость пакета.
type Dependency struct {
	Source  string `yaml:"source,omitempty"`
	Package string `yaml:"package,omitempty"`
	Version string `yaml:"version,omitempty"`
}
