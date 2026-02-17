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

func NewManager() managers.DownloadManager {
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
	if resp, err = m.httpClient.Do(req); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to download file: %w"), err)
	}
	defer resp.Body.Close()

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
	defer file.Close()

	if _, err = io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to write file: %w"), err)
	}

	return
}

func (m *manager) DownloadWithProgress(ctx context.Context, url string, destination string, progress chan<- int) (err error) {

	if strings.HasPrefix(url, protocolFile) {
		path := strings.TrimPrefix(url, protocolFile)
		return m.copyFile(path, destination)
	}

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	var resp *http.Response
	if resp, err = m.httpClient.Do(req); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to download file: %w"), err)
	}
	defer resp.Body.Close()

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
	defer file.Close()

	totalSize := resp.ContentLength
	downloaded := int64(0)

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var n int
		var readErr error
		if n, readErr = resp.Body.Read(buf); n > 0 {
			var writeErr error
			if _, writeErr = file.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf(i18n.Msg("Failed to write file: %w"), writeErr)
			}
			downloaded += int64(n)

			if totalSize > 0 && progress != nil {
				percent := int((downloaded * 100) / totalSize)
				select {
				case progress <- percent:
				default:
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf(i18n.Msg("Read error: %w"), readErr)
		}
	}

	if progress != nil {
		select {
		case progress <- 100:
		default:
		}
		close(progress)
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
	defer sourceFile.Close()

	if destFile, err = os.Create(dst); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "destination file", err)
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
	}

	return
}
