// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

const (
	ManifestFileName     = "manifest.yml"
	ManifestFileNameYAML = "manifest.yaml"
	ReleasesDownloadPath = "/releases/download/"
	ConfigFileName       = "config.yml"
	PackagesDBFileName   = "packages.yml"
	ScopesDirName        = "scopes"
	BinDirName           = "bin"
	LibDirName           = "lib"
	EtcDirName           = "etc"
	CatalogDirName       = "catalog"
	InstalledDirName     = "installed"
	CacheDirName         = "cache"

	FilePermDir  = 0755
	FilePermFile = 0600

	DefaultHomeDir     = "~/.tg"
	HomeDirName        = ".tg"
	URLSchemeFile      = "file"
	URLSchemeHTTPS     = "https"
	URLSchemeHTTP      = "http"
	PathSeparator      = "/"
	URLSchemeSeparator = "://"

	GitHubHost          = "github.com"
	GitHubHostSuffix    = ".github.com"
	ManifestFileExtYAML = ".yaml"
	ManifestFileExtYML  = ".yml"
	ManifestFileExtJSON = ".json"
)

var (
	unsafeChars = map[rune]bool{
		':':  true,
		'?':  true,
		'&':  true,
		'=':  true,
		'%':  true,
		'*':  true,
		'<':  true,
		'>':  true,
		'|':  true,
		'"':  true,
		'\'': true,
	}
)
