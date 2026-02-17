// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"net/http"
)

// Client представляет клиент для работы с GitHub репозиторием.
type Client struct {
	repo       string
	owner      string
	httpClient *http.Client
}

// PluginInfo содержит информацию о плагине из репозитория.
type PluginInfo struct {
	Name     string
	Versions []VersionInfo
}

// VersionInfo содержит информацию о версии плагина.
type VersionInfo struct {
	Version string
	Tag     string
}

// PluginManifest представляет упрощенный манифест плагина из релиза.
// Содержит только базовые поля, необходимые для установки плагина.
// Это общая концепция для всех источников плагинов (GitHub, GitLab и т.д.).
type PluginManifest struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	License     string `json:"license"`
	Description string `json:"description"`
}
