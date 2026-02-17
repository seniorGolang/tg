// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/goccy/go-json"
)

const (
	maxConcurrentDownloads = 5
)

func (c *Client) ListPlugins(ctx context.Context) (plugins []PluginInfo, err error) {

	var latestTag string
	if latestTag, err = c.FindLatestVersionTag(ctx); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to find latest release: %w"), err)
	}

	slog.Debug(i18n.Msg("Latest tag found"), "tag", latestTag, "repo", fmt.Sprintf("%s/%s", c.owner, c.repo))

	var manifestPaths []string
	if manifestPaths, err = c.DownloadManifest(ctx, latestTag); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to download manifest.json: %w"), err)
	}

	slog.Debug(i18n.Msg("manifest.json downloaded"), "count", len(manifestPaths), "tag", latestTag)

	if len(manifestPaths) == 0 {
		return nil, fmt.Errorf(i18n.Msg("manifest.json is empty, no plugins found in tag %s"), latestTag)
	}

	baseURL := fmt.Sprintf(GitHubReleasesBaseURL, c.owner, c.repo, latestTag)
	pluginsMap := make(map[string][]VersionInfo)

	sem := make(chan struct{}, maxConcurrentDownloads)
	type result struct {
		manifest PluginManifest
		err      error
	}
	results := make(chan result, len(manifestPaths))

	// Для каждого пути из manifest.json скачиваем соответствующий манифест параллельно
	for _, manifestPath := range manifestPaths {
		go func(path string) {
			sem <- struct{}{}        // занимаем слот
			defer func() { <-sem }() // освобождаем слот

			// Скачиваем манифест плагина
			// Убираем ведущий слэш, если есть
			cleanPath := strings.TrimPrefix(path, "/")
			manifestURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)
			slog.Debug(i18n.Msg("Downloading plugin manifest"), "path", path, "url", manifestURL)
			var downloadErr error
			var manifest PluginManifest
			if manifest, downloadErr = c.downloadPluginManifest(ctx, manifestURL); downloadErr != nil {
				results <- result{manifest: PluginManifest{}, err: downloadErr}
			} else {
				results <- result{manifest: manifest, err: nil}
			}
		}(manifestPath)
	}

	// Собираем результаты
	var downloadErrors []error
	successCount := 0
	for i := 0; i < len(manifestPaths); i++ {
		res := <-results
		if res.err != nil {
			// Сохраняем ошибки для логирования
			downloadErrors = append(downloadErrors, res.err)
			slog.Debug(i18n.Msg("Failed to download plugin manifest"), "error", res.err)
			continue
		}

		// Извлекаем версию из тега
		var version string
		if version, err = parseTag(latestTag); err != nil {
			slog.Debug(i18n.Msg("Failed to parse tag"), "tag", latestTag, "error", err)
			continue
		}

		// Добавляем информацию о плагине
		pluginsMap[res.manifest.Name] = append(pluginsMap[res.manifest.Name], VersionInfo{
			Version: version,
			Tag:     latestTag,
		})
		successCount++
	}

	// Если ни один манифест не удалось скачать, возвращаем ошибку
	if successCount == 0 {
		if len(downloadErrors) > 0 {
			return nil, fmt.Errorf(i18n.Msg("Failed to download any plugin manifest from %d attempts. First error: %w"), len(manifestPaths), downloadErrors[0])
		}
		return nil, fmt.Errorf(i18n.Msg("Failed to process any plugin from %d manifests"), len(manifestPaths))
	}

	// Логируем предупреждение, если не все манифесты удалось скачать
	if len(downloadErrors) > 0 {
		slog.Warn(i18n.Msg("Some plugin manifests failed to download"), "failed", len(downloadErrors), "total", len(manifestPaths), "success", successCount)
	}

	// Преобразуем в список и сортируем версии
	plugins = make([]PluginInfo, 0, len(pluginsMap))
	for name, versions := range pluginsMap {
		// Сортируем версии по убыванию (последняя версия первая)
		sort.Slice(versions, func(i, j int) bool {
			return compareVersions(versions[i].Version, versions[j].Version) > 0
		})

		plugins = append(plugins, PluginInfo{
			Name:     name,
			Versions: versions,
		})
	}

	return plugins, nil
}

func (c *Client) ListVersions(ctx context.Context, pluginName string) (versions []VersionInfo, err error) {

	var tags []string
	if tags, err = c.listTags(ctx); err != nil {
		return nil, err
	}

	var versionTags []string
	for _, tag := range tags {
		if _, parseErr := parseTag(tag); parseErr == nil {
			versionTags = append(versionTags, tag)
		}
	}

	sort.Slice(versionTags, func(i, j int) bool {
		v1, _ := parseTag(versionTags[i])
		v2, _ := parseTag(versionTags[j])
		return compareVersions(v1, v2) > 0
	})

	versions = make([]VersionInfo, 0)
	for _, tag := range versionTags {
		version, _ := parseTag(tag)

		var manifestPaths []string
		if manifestPaths, err = c.DownloadManifest(ctx, tag); err != nil {
			continue
		}

		found := false
		for _, manifestPath := range manifestPaths {
			baseName := strings.TrimSuffix(filepath.Base(manifestPath), ".json")
			if strings.EqualFold(baseName, pluginName) {
				found = true
				break
			}
		}

		if found {
			versions = append(versions, VersionInfo{
				Version: version,
				Tag:     tag,
			})
		}
	}

	return
}

// FindLatestVersionTag находит последний тег формата v{version}.
func (c *Client) FindLatestVersionTag(ctx context.Context) (tag string, err error) {

	var tags []string
	if tags, err = c.listTags(ctx); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to get list of tags: %w"), err)
	}

	if len(tags) == 0 {
		return "", fmt.Errorf(i18n.Msg("No tags found in repository %s/%s"), c.owner, c.repo)
	}

	slog.Debug(i18n.Msg("Tags found in repository"), "count", len(tags), "repo", fmt.Sprintf("%s/%s", c.owner, c.repo))

	var versionTags []string
	for _, tagItem := range tags {
		if _, parseErr := parseTag(tagItem); parseErr == nil {
			versionTags = append(versionTags, tagItem)
		} else {
			slog.Debug(i18n.Msg("Tag does not match v{version} format"), "tag", tagItem, "error", parseErr)
		}
	}

	if len(versionTags) == 0 {
		return "", fmt.Errorf(i18n.Msg("No tags found in v{version} format in repository %s/%s (found tags: %d)"), c.owner, c.repo, len(tags))
	}

	slog.Debug(i18n.Msg("Tags found in v{version} format"), "count", len(versionTags))

	sort.Slice(versionTags, func(i, j int) bool {
		v1, _ := parseTag(versionTags[i])
		v2, _ := parseTag(versionTags[j])
		return compareVersions(v1, v2) > 0
	})

	tag = versionTags[0]
	return
}

const (
	bodyPreviewLimit = 200
)

// downloadPluginManifest скачивает манифест плагина и возвращает PluginManifest.
func (c *Client) downloadPluginManifest(ctx context.Context, manifestURL string) (manifest PluginManifest, err error) {

	var body []byte
	if body, err = c.DownloadManifestContent(ctx, manifestURL); err != nil {
		return PluginManifest{}, fmt.Errorf(i18n.Msg("Failed to download manifest: %w"), err)
	}

	bodyStr := string(body)
	bodyPreview := bodyStr
	if len(bodyPreview) > bodyPreviewLimit {
		bodyPreview = bodyPreview[:bodyPreviewLimit] + "..."
	}
	slog.Debug(i18n.Msg("Plugin manifest downloaded"), "url", manifestURL, "size", len(body), "preview", bodyPreview)

	if len(body) == 0 {
		return PluginManifest{}, fmt.Errorf(i18n.Msg("Plugin manifest is empty: %s"), manifestURL)
	}

	var pluginManifest PluginManifest
	if err = json.Unmarshal(body, &pluginManifest); err != nil {
		slog.Debug(i18n.Msg("Error parsing manifest"), "url", manifestURL, "body", bodyStr, "error", err)
		return PluginManifest{}, fmt.Errorf(i18n.Msg("Failed to parse manifest: %w"), err)
	}

	manifest = pluginManifest
	return
}
