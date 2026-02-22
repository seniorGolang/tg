// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package file

import (
	"context"
	"net/url"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

const (
	distName = "file"
)

// Dist реализует Dist для file:// источников.
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

	return parsedURL.Scheme == storage.URLSchemeFile
}

// GetVersions: для file:// версии не поддерживаются.
func (d *Dist) GetVersions(ctx context.Context, source string) (versions []string, err error) {

	return []string{}, nil
}

func (d *Dist) ManifestURL(ctx context.Context, source string, version string) (manifestURL string, err error) {

	if strings.HasSuffix(source, storage.ManifestFileExtYAML) ||
		strings.HasSuffix(source, storage.ManifestFileExtYML) ||
		strings.HasSuffix(source, storage.ManifestFileExtJSON) {
		return source, nil
	}

	return trimTrailingSeparator(source) + storage.PathSeparator + storage.ManifestFileName, nil
}

// FileURL формирует URL для загрузки файла из file:// источника.
// Возвращает относительный путь от source.
func (d *Dist) FileURL(source string, version string, filename string) (fileURL string, err error) {

	if strings.HasPrefix(filename, "/") || strings.Contains(filename, "://") {
		fileURL = filename
		return
	}

	return trimTrailingSeparator(source) + storage.PathSeparator + filename, nil
}

func trimTrailingSeparator(path string) (trimmed string) {

	if path == "" {
		return ""
	}
	return strings.TrimSuffix(path, storage.PathSeparator)
}
