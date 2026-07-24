// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

type Manifest struct {
	Version   string        `yaml:"version"`
	Manifests []ManifestRef `yaml:"manifests,omitempty"`
	Packages  []Package     `yaml:"packages,omitempty"`
}

type ManifestRef struct {
	URL string `yaml:"url"`
}

type Package struct {
	Name         string             `yaml:"name"`
	Descr        string             `yaml:"descr,omitempty"`
	Hidden       bool               `yaml:"hidden,omitempty"`
	Alias        string             `yaml:"alias,omitempty"`
	Downloads    []PlatformDownload `yaml:"downloads"`
	Files        []FileInstallation `yaml:"files"`
	Skills       []SkillSpec        `yaml:"skills,omitempty"`
	Scripts      *Scripts           `yaml:"scripts,omitempty"`
	Dependencies []string           `yaml:"dependencies,omitempty"`
}

// SkillSpec описывает skill в манифесте пакета (root относительно InstallPrefix).
type SkillSpec struct {
	Name string `yaml:"name"`
	Root string `yaml:"root"`
}

type PlatformDownload struct {
	OS   string `yaml:"os,omitempty"`
	Arch string `yaml:"arch,omitempty"`
	URL  string `yaml:"url"`
}

type FileInstallation struct {
	File        string `yaml:"file,omitempty"`
	Source      string `yaml:"source,omitempty"`
	Destination string `yaml:"destination"`
	Checksum    string `yaml:"checksum,omitempty"`
}

type Scripts struct {
	PreInstall    *ScriptAction `yaml:"pre_install,omitempty"`
	PostInstall   *ScriptAction `yaml:"post_install,omitempty"`
	PreUninstall  *ScriptAction `yaml:"pre_uninstall,omitempty"`
	PostUninstall *ScriptAction `yaml:"post_uninstall,omitempty"`
}

type ScriptAction struct {
	Script string `yaml:"script,omitempty"`
	Source string `yaml:"source,omitempty"`
	Exec   string `yaml:"exec"`
}

type Dependency struct {
	Source  string `yaml:"source,omitempty"`
	Package string `yaml:"package,omitempty"`
	Version string `yaml:"version,omitempty"`
}
