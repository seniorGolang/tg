// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/installer/github"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	ver "github.com/seniorGolang/tg/v3/internal/installer/version"
)

const (
	distName = "github"
)

// Dist реализует Dist для GitHub источников.
type Dist struct{}

func NewDist() (d *Dist) {

	return &Dist{}
}

func (d *Dist) Name() (name string) {

	return distName
}

func (d *Dist) IsMine(urlStr string) (isMine bool) {

	var parsedURL *url.URL
	var err error
	if parsedURL, err = url.Parse(urlStr); err != nil {
		return false
	}

	if parsedURL.Scheme == "" {
		// Пробуем добавить схему
		testURL := "https://" + urlStr
		if parsedURL, err = url.Parse(testURL); err != nil {
			return false
		}
	}

	isMine = parsedURL.Host == storage.GitHubHost() || strings.HasSuffix(parsedURL.Host, storage.GitHubHostSuffix())
	return
}

// GetVersions использует FindLatestVersionTag (listTags приватный).
// Для получения всех версий можно расширить github.Client.
func (d *Dist) GetVersions(ctx context.Context, source string) (versions []string, err error) {

	var githubClient *github.Client
	if githubClient, err = github.NewClient(source); err != nil {
		err = fmt.Errorf("failed to create GitHub client: %w", err)
		return
	}

	var latestTag string
	if latestTag, err = githubClient.FindLatestVersionTag(ctx); err != nil {
		err = fmt.Errorf("failed to find latest version tag: %w", err)
		return
	}

	// Убираем префикс "v" из тега
	versionStr := strings.TrimPrefix(latestTag, ver.VersionPrefix)
	versions = []string{versionStr}

	return
}

// ManifestURL формирует URL манифеста для GitHub репозитория.
func (d *Dist) ManifestURL(ctx context.Context, source string, version string) (manifestURL string, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(source); err != nil {
		err = fmt.Errorf("failed to parse source URL: %w", err)
		return
	}

	if !d.IsMine(source) {
		err = fmt.Errorf("source is not a GitHub URL")
		return
	}

	// Если version пустая или "latest", получаем последнюю версию
	if version == "" || version == ver.LatestVersion {
		var versions []string
		if versions, err = d.GetVersions(ctx, source); err != nil {
			err = fmt.Errorf("failed to get versions: %w", err)
			return
		}

		if len(versions) == 0 {
			err = fmt.Errorf("no versions available")
			return
		}

		// Берем первую версию (GetVersions возвращает последнюю)
		version = versions[0]
	}

	pathParts := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		err = fmt.Errorf("invalid GitHub URL format: expected owner/repo")
		return
	}

	owner := pathParts[0]
	repo := pathParts[1]

	tag := ensureVersionPrefix(version)

	releasesPath := strings.TrimPrefix(storage.ReleasesDownloadPath, storage.PathSeparator)
	releasesPath = strings.TrimSuffix(releasesPath, storage.PathSeparator)
	manifestPath := strings.Join([]string{
		owner,
		repo,
		releasesPath,
		tag,
		storage.ManifestFileName,
	}, storage.PathSeparator)

	manifestURLObj := &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   storage.PathSeparator + manifestPath,
	}

	manifestURL = manifestURLObj.String()
	return
}

// FileURL формирует URL для загрузки файла из GitHub releases.
func (d *Dist) FileURL(source string, version string, filename string) (fileURL string, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(source); err != nil {
		err = fmt.Errorf("failed to parse source URL: %w", err)
		return
	}

	if !d.IsMine(source) {
		err = fmt.Errorf("source is not a GitHub URL")
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		err = fmt.Errorf("invalid GitHub URL format: expected owner/repo")
		return
	}

	owner := pathParts[0]
	repo := pathParts[1]

	tag := ensureVersionPrefix(version)

	releasesPath := strings.TrimPrefix(storage.ReleasesDownloadPath, storage.PathSeparator)
	releasesPath = strings.TrimSuffix(releasesPath, storage.PathSeparator)
	filePath := strings.Join([]string{
		owner,
		repo,
		releasesPath,
		tag,
		filename,
	}, storage.PathSeparator)

	fileURLObj := &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   storage.PathSeparator + filePath,
	}

	fileURL = fileURLObj.String()
	return
}

// ensureVersionPrefix добавляет префикс версии, если его нет.
func ensureVersionPrefix(version string) (tag string) {

	tag = version
	if !strings.HasPrefix(tag, ver.VersionPrefix) {
		tag = ver.VersionPrefix + tag
	}

	return
}
