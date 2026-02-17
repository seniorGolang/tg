// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installer

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/installer/github"
)

// PluginInfo содержит информацию о плагине из репозитория.
type PluginInfo = github.PluginInfo

// VersionInfo содержит информацию о версии плагина.
type VersionInfo = github.VersionInfo

// PluginManifest представляет упрощенный манифест плагина из релиза.
// Содержит только базовые поля, необходимые для установки плагина.
// Это общая концепция для всех источников плагинов (GitHub, GitLab и т.д.).
// Экспортируется через алиас из github для избежания циклических зависимостей.
type PluginManifest = github.PluginManifest

// PluginSource представляет интерфейс для источника плагинов.
// Позволяет абстрагироваться от конкретной реализации (GitHub, GitLab и т.д.).
type PluginSource interface {
	// ListPlugins возвращает список всех доступных плагинов в репозитории.
	ListPlugins(ctx context.Context) (plugins []PluginInfo, err error)

	// ListVersions возвращает список версий конкретного плагина.
	ListVersions(ctx context.Context, pluginName string) (versions []VersionInfo, err error)

	// DownloadPluginFiles скачивает все необходимые файлы плагина и записывает их напрямую на диск.
	// installPath - относительный путь от projectRoot (например, "plugins/test/1.2.10").
	// Возвращает относительные пути к сохранённым файлам и размеры файлов.
	DownloadPluginFiles(ctx context.Context, pluginName, version, installPath string) (
		jsonPath string, jsonSize int64, tgpPath string, tgpSize int64,
		sha256Path string, sha256Size int64, err error)
}
