// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

type ScopeInfo struct {
	Name          string
	IsActive      bool
	PackageCount  int
	ManifestCount int
}

type ScopeConfig struct {
	Name          string
	BinDir        string
	LibDir        string
	ConfigDir     string
	InstallPrefix string
}
