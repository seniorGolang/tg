// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

// ScopeInfo представляет информацию о scope.
type ScopeInfo struct {
	Name          string
	IsActive      bool
	PackageCount  int
	ManifestCount int
}

// ScopeConfig представляет конфигурацию scope.
type ScopeConfig struct {
	Name          string
	InstallPrefix string
	BinDir        string
	LibDir        string
	ConfigDir     string
}

// ScopeOptions представляет опции для создания scope.
type ScopeOptions struct {
	From   string
	Config string
}
