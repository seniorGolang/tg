// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
)

const (
	protocolFile = "file://"
)

type manager struct {
	httpClient *http.Client
}

func NewManager() (m managers.DownloadManager) {
	return &manager{
		httpClient: &http.Client{},
	}
}

func (m *manager) Download(ctx context.Context, url string, destination string) (err error) {

	if strings.HasPrefix(url, protocolFile) {
		path := strings.TrimPrefix(url, protocolFile)
		return m.copyFile(path, destination)
	}

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	var resp *http.Response
	//nolint:gosec // G704: URL из манифеста/конфига, валидация на уровне вызывающего кода
	if resp, err = m.httpClient.Do(req); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to download file: %w"), err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(i18n.Msg("Unexpected status code: %d"), resp.StatusCode)
	}

	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
	}

	var file *os.File
	if file, err = os.Create(destination); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
	}
	defer func() { _ = file.Close() }()

	if _, err = io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to write file: %w"), err)
	}

	return
}

func (m *manager) DownloadWithProgress(ctx context.Context, url string, destination string, onProgress func(percent int)) (err error) {

	if strings.HasPrefix(url, protocolFile) {
		path := strings.TrimPrefix(url, protocolFile)
		return m.copyFile(path, destination)
	}

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	var resp *http.Response
	//nolint:gosec // G704: URL из манифеста/конфига, валидация на уровне вызывающего кода
	if resp, err = m.httpClient.Do(req); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to download file: %w"), err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(i18n.Msg("Unexpected status code: %d"), resp.StatusCode)
	}

	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
	}

	var file *os.File
	if file, err = os.Create(destination); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
	}
	fileClosed := false
	defer func() {
		if !fileClosed {
			_ = file.Close()
		}
	}()

	var writer io.Writer = file
	if onProgress != nil && resp.ContentLength > 0 {
		writer = &progressWriter{
			dst:        file,
			total:      resp.ContentLength,
			onProgress: onProgress,
		}
	}

	var written int64
	if written, err = io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to write file: %w"), err)
	}

	if err = file.Close(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to close file: %w"), err)
	}
	fileClosed = true

	if resp.ContentLength > 0 && written != resp.ContentLength {
		return fmt.Errorf(i18n.Msg("File size on disk (%d) does not match Content-Length (%d)"), written, resp.ContentLength)
	}

	if onProgress != nil {
		onProgress(100)
	}

	return
}

type progressWriter struct {
	dst        io.Writer
	total      int64
	written    int64
	onProgress func(percent int)
}

func (w *progressWriter) Write(p []byte) (n int, err error) {

	if n, err = w.dst.Write(p); n > 0 {
		w.written += int64(n)
		if w.total > 0 && w.onProgress != nil {
			w.onProgress(int((w.written * 100) / w.total))
		}
	}
	return
}

func (m *manager) copyFile(src string, dst string) (err error) {

	if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
	}

	var destFile *os.File
	var sourceFile *os.File
	if sourceFile, err = os.Open(src); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to open source file: %w"), err)
	}
	defer func() { _ = sourceFile.Close() }()

	if destFile, err = os.Create(dst); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "destination file", err)
	}
	defer func() { _ = destFile.Close() }()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
	}

	return
}
