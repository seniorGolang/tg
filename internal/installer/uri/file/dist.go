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

	return distName
}

func (d *Dist) IsMine(urlStr string) (isMine bool) {

	var parsedURL *url.URL
	var err error
	if parsedURL, err = url.Parse(urlStr); err != nil {
		return false
	}

	isMine = parsedURL.Scheme == storage.URLSchemeFile
	return
}

// GetVersions: для file:// версии не поддерживаются.
func (d *Dist) GetVersions(ctx context.Context, source string) (versions []string, err error) {

	// file:// URL не поддерживают получение версии, возвращаем пустой массив
	versions = []string{}
	return
}

// ManifestURL формирует URL манифеста для file:// источника.
// version игнорируется для file:// URL.
func (d *Dist) ManifestURL(ctx context.Context, source string, version string) (manifestURL string, err error) {

	// Если source уже указывает на файл манифеста, возвращаем его
	if strings.HasSuffix(source, storage.ManifestFileExtYAML()) ||
		strings.HasSuffix(source, storage.ManifestFileExtYML()) ||
		strings.HasSuffix(source, storage.ManifestFileExtJSON()) {
		manifestURL = source
		return
	}

	// Иначе добавляем имя файла манифеста
	manifestURL = trimTrailingSeparator(source) + storage.PathSeparator + storage.ManifestFileName
	return
}

// FileURL формирует URL для загрузки файла из file:// источника.
// Возвращает относительный путь от source.
func (d *Dist) FileURL(source string, version string, filename string) (fileURL string, err error) {

	// Если filename уже полный путь или URL, возвращаем его
	if strings.HasPrefix(filename, "/") || strings.Contains(filename, "://") {
		fileURL = filename
		return
	}

	// Иначе формируем путь относительно source
	fileURL = trimTrailingSeparator(source) + storage.PathSeparator + filename
	return
}

// trimTrailingSeparator убирает завершающий разделитель пути.
func trimTrailingSeparator(path string) (trimmed string) {

	trimmed = strings.TrimSuffix(path, storage.PathSeparator)
	return
}
