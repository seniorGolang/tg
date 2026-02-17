// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

const (
	distName = "proxy"
)

// Dist реализует Dist для proxy источников.
type Dist struct{}

func NewDist() (d *Dist) {
	return &Dist{}
}

func (d *Dist) Name() (name string) {

	if d == nil {
		return ""
	}
	return distName
}

func (d *Dist) IsMine(urlStr string) (isMine bool) {

	// Proxy определяется по наличию пути в URL
	var err error
	var parsedURL *url.URL
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

	// Proxy определяется по наличию пути (не file:// и не GitHub)
	if parsedURL.Scheme == storage.URLSchemeFile {
		return false
	}

	if parsedURL.Host == storage.GitHubHost || strings.HasSuffix(parsedURL.Host, storage.GitHubHostSuffix) {
		return false
	}

	return true
}

func (d *Dist) GetVersions(ctx context.Context, source string) (versions []string, err error) {

	var base *url.URL
	if base, err = d.requestBaseURL(source); err != nil {
		err = fmt.Errorf("request base URL: %w", err)
		return
	}

	versionsURL := joinPathToURL(base, "versions")

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, versionsURL, nil); err != nil {
		err = fmt.Errorf("failed to create HTTP request: %w", err)
		return
	}

	client := &http.Client{}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		err = fmt.Errorf("failed to execute HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return
	}

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		return
	}

	if err = json.Unmarshal(body, &versions); err != nil {
		err = fmt.Errorf("failed to parse JSON response: %w", err)
		return
	}

	if len(versions) == 0 {
		err = fmt.Errorf("empty versions list")
		return
	}

	return
}

// ManifestURL формирует URL манифеста для формата tg-proxy.
func (d *Dist) ManifestURL(ctx context.Context, source string, versionStr string) (manifestURL string, err error) {

	var base *url.URL
	if base, err = d.requestBaseURL(source); err != nil {
		err = fmt.Errorf("request base URL: %w", err)
		return
	}

	// Если version пустая или "latest", получаем последнюю версию
	if versionStr == "" || versionStr == version.LatestVersion {
		var versions []string
		if versions, err = d.GetVersions(ctx, source); err != nil {
			err = fmt.Errorf("get versions: %w", err)
			return
		}

		if len(versions) == 0 {
			err = fmt.Errorf("no versions available")
			return
		}

		parsedVersions := make([]models.Version, 0, len(versions))
		for _, v := range versions {
			var parsedVersion models.Version
			if parsedVersion, err = version.Parse(v); err != nil {
				continue
			}
			parsedVersions = append(parsedVersions, parsedVersion)
		}

		if len(parsedVersions) == 0 {
			err = fmt.Errorf("no valid versions found")
			return
		}

		versionStr = version.Latest(parsedVersions).Original
	}

	manifestURL = joinPathToURL(base, versionStr, storage.ManifestFileName)
	return
}

// FileURL формирует URL для загрузки файла из proxy.
func (d *Dist) FileURL(source string, version string, filename string) (fileURL string, err error) {

	var base *url.URL
	if base, err = d.requestBaseURL(source); err != nil {
		err = fmt.Errorf("request base URL: %w", err)
		return
	}

	fileURL = joinPathToURL(base, version, filename)
	return
}

// joinPathToURL присоединяет сегменты пути к базовому URL через path.Join и возвращает полный URL.
func joinPathToURL(base *url.URL, segments ...string) (fullURL string) {

	u := *base
	u.Path = path.Join(append([]string{u.Path}, segments...)...)
	return u.String()
}

// requestBaseURL: из последнего сегмента пути убирается суффикс ":version" при наличии.
func (d *Dist) requestBaseURL(source string) (base *url.URL, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(source); err != nil {
		err = fmt.Errorf("parse source URL: %w", err)
		return
	}
	base = &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   parsedURL.Path,
	}
	return
}
