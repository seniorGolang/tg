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
	ReplacementChar    = "_"

	UnsafeCharColon       = ":"
	UnsafeCharQuestion    = "?"
	UnsafeCharAmpersand   = "&"
	UnsafeCharEquals      = "="
	UnsafeCharPercent     = "%"
	UnsafeCharAsterisk    = "*"
	UnsafeCharLessThan    = "<"
	UnsafeCharGreaterThan = ">"
	UnsafeCharPipe        = "|"
	UnsafeCharDoubleQuote = "\""
	UnsafeCharSingleQuote = "'"
	UnsafeCharDot         = "."
	UnsafeCharDash        = "-"

	githubHost          = "github.com"
	githubHostSuffix    = ".github.com"
	manifestFileExtYAML = ".yaml"
	manifestFileExtYML  = ".yml"
	manifestFileExtJSON = ".json"
	versionPrefix       = "v"
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

func GitHubHost() (host string) {

	return githubHost
}

func GitHubHostSuffix() (suffix string) {

	return githubHostSuffix
}

func VersionPrefix() (prefix string) {

	return versionPrefix
}

func ManifestFileExtYAML() (ext string) {

	return manifestFileExtYAML
}

func ManifestFileExtYML() (ext string) {

	return manifestFileExtYML
}

func ManifestFileExtJSON() (ext string) {

	return manifestFileExtJSON
}
