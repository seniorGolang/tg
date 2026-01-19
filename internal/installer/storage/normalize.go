// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"net/url"
	"path/filepath"
	"strings"
)

func NormalizeSource(sourceURL string) (normalized string) {

	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		return sanitizePath(sourceURL)
	}

	if parsedURL.Scheme == URLSchemeFile {
		path := parsedURL.Path
		path = strings.TrimPrefix(path, PathSeparator)
		return sanitizePath(path)
	}

	parts := make([]string, 0, 4)
	if parsedURL.Scheme != "" {
		parts = append(parts, parsedURL.Scheme)
	}
	if parsedURL.Host != "" {
		parts = append(parts, parsedURL.Host)
	}

	path := strings.TrimPrefix(parsedURL.Path, PathSeparator)
	if path != "" {
		for _, part := range strings.Split(path, PathSeparator) {
			if part != "" {
				parts = append(parts, sanitizePathComponent(part))
			}
		}
	}

	if len(parts) == 0 {
		return sanitizePath(sourceURL)
	}

	return filepath.Join(parts...)
}

func baseSourceURL(anyURL string) (source string) {

	parsedURL, err := url.Parse(anyURL)
	if err != nil {
		return anyURL
	}

	if parsedURL.Scheme == URLSchemeFile {
		path := parsedURL.Path
		path = removeFileNameFromPath(path)
		path = strings.TrimSuffix(path, ReleasesDownloadPath)
		path = strings.TrimSuffix(path, PathSeparator)
		idx := strings.LastIndex(path, PathSeparator)
		if idx > 0 {
			path = path[:idx]
		}
		u := &url.URL{Scheme: URLSchemeFile, Path: path}
		return u.String()
	}

	path := parsedURL.Path
	path = removeFileNameFromPath(path)
	if strings.Contains(path, ReleasesDownloadPath) {
		if idx := strings.Index(path, ReleasesDownloadPath); idx >= 0 {
			path = path[:idx]
		}
	}
	u := &url.URL{Scheme: parsedURL.Scheme, Host: parsedURL.Host, Path: path}
	return u.String()
}

func ExtractSourceFromManifestURL(manifestURL string) (source string) {

	return baseSourceURL(manifestURL)
}

// Пути без схемы (один сегмент host или host/path) — старый формат, восстанавливаем как https.
func ExtractSourceFromNormalizedPath(normalizedPath string) (source string) {

	return extractURLFromNormalizedPath(normalizedPath)
}

// extractURLFromNormalizedPath восстанавливает URL из пути, сохранённого через filepath.Join (scheme/host/path).
// Если первый сегмент не http(s) — считаем путь файловым (file://). Иначе: scheme/host/path → scheme://host/path или один host → https://host.
func extractURLFromNormalizedPath(normalizedPath string) (source string) {

	parts := strings.Split(normalizedPath, string(filepath.Separator))
	if len(parts) == 0 {
		return normalizedPath
	}

	if parts[0] != URLSchemeHTTP && parts[0] != URLSchemeHTTPS {
		pathWithSlash := strings.ReplaceAll(normalizedPath, string(filepath.Separator), PathSeparator)
		return URLSchemeFile + URLSchemeSeparator + PathSeparator + pathWithSlash
	}

	var scheme, host string
	var pathParts []string

	switch {
	case len(parts) >= 2 && (parts[0] == URLSchemeHTTP || parts[0] == URLSchemeHTTPS):
		scheme = parts[0]
		host = parts[1]
		pathParts = parts[2:]
	default:
		// Один сегмент (host) — восстанавливаем как https://host
		scheme = URLSchemeHTTPS
		host = parts[0]
		pathParts = parts[1:]
	}

	u := &url.URL{Scheme: scheme, Host: host}
	if len(pathParts) > 0 {
		u.Path = PathSeparator + strings.Join(pathParts, PathSeparator)
	}
	return u.String()
}

func NormalizeSourceForInstallation(sourceURL string) (normalizedSource string) {

	return baseSourceURL(sourceURL)
}
