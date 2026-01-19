// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/pterm/pterm"
)

const (
	maxRedirects = 10
	dirPerm      = 0755
)

// DownloadFile скачивает файл по URL с обработкой редиректов и отображением прогресса.
// Это единая функция для загрузки всех типов файлов (manifest.json, plugin.json, .tgp, .sha256).
// url - URL файла для скачивания
// filePath - путь для сохранения файла
// showProgress - показывать ли прогресс-бар (true для больших файлов)
// Возвращает размер скачанного файла
func (c *Client) DownloadFile(ctx context.Context, url string, filePath string, showProgress bool) (size int64, err error) {

	return c.downloadFile(ctx, url, filePath, 0, showProgress)
}

// downloadFile скачивает файл с обработкой редиректов и отображением прогресса.
func (c *Client) downloadFile(ctx context.Context, url string, filePath string, redirectCount int, showProgress bool) (size int64, err error) {

	filePath = filepath.Clean(filePath)
	if redirectCount >= maxRedirects {
		return 0, fmt.Errorf(i18n.Msg("Maximum number of redirects exceeded: %d"), maxRedirects)
	}

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	req.Header.Set("Accept", "application/octet-stream")

	var resp *http.Response
	if resp, err = c.httpClient.Do(req); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to download file: %w"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		resp.Body.Close()
		location := resp.Header.Get("Location")
		if location == "" {
			return 0, fmt.Errorf(i18n.Msg("Redirect without Location header: %s"), url)
		}
		return c.downloadFile(ctx, location, filePath, redirectCount+1, showProgress)
	}

	if resp.StatusCode != http.StatusOK {
		var body []byte
		var readErr error
		if body, readErr = io.ReadAll(resp.Body); readErr != nil {
			return 0, fmt.Errorf(i18n.Msg("Failed to read response body: %w"), readErr)
		}

		if resp.StatusCode == http.StatusNotFound {
			return 0, fmt.Errorf(i18n.Msg("File not found (404): %s"), url)
		}

		if resp.StatusCode == http.StatusForbidden {
			return 0, fmt.Errorf("%s", i18n.Msg("Access forbidden (403)"))
		}

		return 0, fmt.Errorf(i18n.Msg("Unexpected status code: %d, body: %s"), resp.StatusCode, string(body))
	}

	dir := filepath.Clean(filepath.Dir(filePath))
	if err = os.MkdirAll(dir, dirPerm); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to create directory %s: %w"), dir, err)
	}

	var file *os.File
	if file, err = os.Create(filePath); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to create file %s: %w"), filePath, err)
	}
	defer file.Close()

	fileName := filepath.Base(filePath)
	totalSize := resp.ContentLength
	var reader io.Reader = resp.Body

	var bar *pterm.ProgressbarPrinter
	if showProgress && totalSize > 0 {
		bar, _ = pterm.DefaultProgressbar.
			WithTotal(100).
			WithTitle(fmt.Sprintf(i18n.Msg("Downloading %s"), fileName)).
			Start()
		defer func() {
			if bar != nil {
				_, _ = bar.Stop()
			}
		}()
	}

	if bar != nil {
		reader = &progressReader{
			reader:     resp.Body,
			total:      totalSize,
			downloaded: 0,
			bar:        bar,
		}
	}

	var bytesWritten int64
	if bytesWritten, err = io.Copy(file, reader); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to write data to file %s: %w"), filePath, err)
	}

	if err = file.Close(); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to close file %s: %w"), filePath, err)
	}

	var fileInfo os.FileInfo
	if fileInfo, err = os.Stat(filePath); err != nil {
		return 0, fmt.Errorf(i18n.Msg("Failed to get file information %s: %w"), filePath, err)
	}

	if fileInfo.Size() != bytesWritten {
		return 0, fmt.Errorf(i18n.Msg("File size on disk (%d) does not match written size (%d)"), fileInfo.Size(), bytesWritten)
	}

	if totalSize > 0 && fileInfo.Size() != totalSize {
		return 0, fmt.Errorf(i18n.Msg("File size on disk (%d) does not match Content-Length (%d)"), fileInfo.Size(), totalSize)
	}

	return fileInfo.Size(), nil
}

// progressReader оборачивает io.Reader и отслеживает прогресс чтения для отображения в прогресс-баре.
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	bar        *pterm.ProgressbarPrinter
}

// Read переопределяет метод Read для отслеживания прогресса.
func (pr *progressReader) Read(p []byte) (n int, err error) {

	n, err = pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)

		// Обновляем прогресс-бар
		if pr.bar != nil && pr.total > 0 {
			percentage := int(float64(pr.downloaded) / float64(pr.total) * 100)
			if percentage > 100 {
				percentage = 100
			}
			// Обновляем прогресс-бар до текущего процента
			current := pr.bar.Current
			if percentage > current {
				for i := current; i < percentage; i++ {
					pr.bar.Increment()
				}
			}
		}
	}
	return n, err
}

const (
	pluginJSONFilename = "plugin.json"
)

// DownloadPluginFiles скачивает все необходимые файлы плагина и записывает их напрямую на диск.
// installPath - относительный путь от projectRoot (например, "plugins/test/1.2.10").
// Возвращает относительные пути к сохранённым файлам и размеры файлов.
func (c *Client) DownloadPluginFiles(ctx context.Context, pluginName string, version string, installPath string) (jsonPath string, jsonSize int64, tgpPath string, tgpSize int64, sha256Path string, sha256Size int64, err error) {

	tag := fmt.Sprintf("%s%s", VersionTagPrefix, version)
	baseURL := fmt.Sprintf(GitHubReleasesBaseURL, c.owner, c.repo, tag)

	jsonURL := fmt.Sprintf("%s/%s.json", baseURL, pluginName)
	tgpURL := fmt.Sprintf("%s/%s.tgp", baseURL, pluginName)
	sha256URL := fmt.Sprintf("%s/%s.sha256", baseURL, pluginName)

	jsonPath = filepath.Join(installPath, pluginJSONFilename)
	tgpPath = filepath.Join(installPath, fmt.Sprintf("%s.tgp", pluginName))
	sha256Path = filepath.Join(installPath, fmt.Sprintf("%s.sha256", pluginName))

	slog.Debug(i18n.Msg("Downloading plugin files"), "plugin", pluginName, "version", version, "json", jsonURL, "tgp", tgpURL)

	if jsonSize, err = c.DownloadFile(ctx, jsonURL, jsonPath, false); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, "", 0, "", 0, fmt.Errorf(i18n.Msg("Failed to download plugin.json from %s: %w"), jsonURL, err)
	}
	slog.Debug(i18n.Msg("plugin.json downloaded"), "size", jsonSize, "path", jsonPath)

	if tgpSize, err = c.DownloadFile(ctx, tgpURL, tgpPath, true); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, "", 0, "", 0, fmt.Errorf(i18n.Msg("Failed to download %s.tgp from %s: %w"), pluginName, tgpURL, err)
	}

	if sha256Size, err = c.DownloadFile(ctx, sha256URL, sha256Path, false); err != nil {
		_ = storage.CleanupInstallPath(installPath)
		return "", 0, "", 0, "", 0, fmt.Errorf(i18n.Msg("Failed to download %s.sha256 from %s: %w"), pluginName, sha256URL, err)
	}

	return
}

// DownloadManifestContent скачивает содержимое файла по URL и возвращает его как байты.
// Используется для скачивания manifest.json и манифестов плагинов.
func (c *Client) DownloadManifestContent(ctx context.Context, url string) (content []byte, err error) {

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	req.Header.Set("Accept", "application/json")

	var resp *http.Response
	if resp, err = c.httpClient.Do(req); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to download file: %w"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		location := resp.Header.Get("Location")
		if location == "" {
			return nil, fmt.Errorf(i18n.Msg("Redirect without Location header: %s"), url)
		}
		return c.DownloadManifestContent(ctx, location)
	}

	if resp.StatusCode != http.StatusOK {
		var body []byte
		var readErr error
		if body, readErr = io.ReadAll(resp.Body); readErr != nil {
			return nil, fmt.Errorf(i18n.Msg("Unexpected status code: %d, failed to read response body: %w"), resp.StatusCode, readErr)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf(i18n.Msg("File not found (404): %s"), url)
		}

		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%s", i18n.Msg("Access forbidden (403)"))
		}

		return nil, fmt.Errorf(i18n.Msg("Unexpected status code: %d, body: %s"), resp.StatusCode, string(body))
	}

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to read response: %w"), err)
	}

	content = body
	return
}
