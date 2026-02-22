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

	var err error
	var parsedURL *url.URL
	if parsedURL, err = url.Parse(urlStr); err != nil {
		return false
	}

	if parsedURL.Scheme == "" {
		testURL := "https://" + urlStr
		if parsedURL, err = url.Parse(testURL); err != nil {
			return false
		}
	}

	// Proxy определяется по наличию пути (не file:// и не GitHub).
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
		return nil, fmt.Errorf("request base URL: %w", err)
	}

	versionsURL := joinPathToURL(base, "versions")

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, versionsURL, nil); err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	client := &http.Client{}
	var resp *http.Response
	//nolint:gosec // G704: URL из конфигурации прокси, валидация на уровне вызывающего кода
	if resp, err = client.Do(req); err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err = json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("empty versions list")
	}

	return
}

func (d *Dist) ManifestURL(ctx context.Context, source string, versionStr string) (manifestURL string, err error) {

	var base *url.URL
	if base, err = d.requestBaseURL(source); err != nil {
		return "", fmt.Errorf("request base URL: %w", err)
	}

	// Если version пустая или "latest", получаем последнюю версию
	if versionStr == "" || versionStr == version.LatestVersion {
		var versions []string
		if versions, err = d.GetVersions(ctx, source); err != nil {
			return "", fmt.Errorf("get versions: %w", err)
		}

		if len(versions) == 0 {
			return "", fmt.Errorf("no versions available")
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
			return "", fmt.Errorf("no valid versions found")
		}

		versionStr = version.Latest(parsedVersions).Original
	}

	return joinPathToURL(base, versionStr, storage.ManifestFileName), nil
}

func (d *Dist) FileURL(source string, version string, filename string) (fileURL string, err error) {

	var base *url.URL
	if base, err = d.requestBaseURL(source); err != nil {
		return "", fmt.Errorf("request base URL: %w", err)
	}

	return joinPathToURL(base, version, filename), nil
}

func joinPathToURL(base *url.URL, segments ...string) (fullURL string) {

	u := *base
	u.Path = path.Join(append([]string{u.Path}, segments...)...)
	return u.String()
}

// requestBaseURL: из последнего сегмента пути убирается суффикс ":version" при наличии.
func (d *Dist) requestBaseURL(source string) (base *url.URL, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(source); err != nil {
		return nil, fmt.Errorf("parse source URL: %w", err)
	}
	return &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   parsedURL.Path,
	}, nil
}
