// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// ParsePluginPath парсит путь к плагину в формате go install.
func ParsePluginPath(pluginPath string) (repoURL string, pluginName string, version string, err error) {

	parts := strings.Split(pluginPath, versionSeparator)
	pathPart := parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(pathPart); err != nil {
		return "", "", "", fmt.Errorf(i18n.Msg("Invalid URL: %w"), err)
	}

	if parsedURL.Host != githubHost {
		return "", "", "", errors.New(i18n.Msg("Only GitHub repositories are supported"))
	}

	urlPath := strings.TrimPrefix(parsedURL.Path, pathPrefix)
	pathSegments := strings.Split(urlPath, pathPrefix)

	if len(pathSegments) < minPathSegments {
		return "", "", "", errors.New(i18n.Msg("Invalid URL format: expected github.com/user/repo"))
	}

	owner := pathSegments[0]
	repo := pathSegments[1]
	repoURL = fmt.Sprintf(githubURLFormat, owner, repo)

	if len(pathSegments) > minPathSegments {
		pluginName = pathSegments[2]
	}

	if version != "" {
		version = strings.TrimPrefix(version, versionPrefix)
	}

	return
}

// NormalizeGitHubURL нормализует GitHub URL, убирая лишние части.
func NormalizeGitHubURL(rawURL string) (normalizedURL string, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(rawURL); err != nil {
		return "", fmt.Errorf(i18n.Msg("Invalid URL: %w"), err)
	}

	if parsedURL.Scheme == "" {
		rawURL = githubScheme + rawURL
		if parsedURL, err = url.Parse(rawURL); err != nil {
			return "", fmt.Errorf(i18n.Msg("Invalid URL: %w"), err)
		}
	}

	parsedURL.Path = path.Clean(parsedURL.Path)
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	normalizedURL = parsedURL.String()
	return
}
