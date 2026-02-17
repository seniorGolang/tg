// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"context"
	"errors"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

// dist представляет источник распространения пакетов.
type dist interface {
	Name() (name string)

	// IsMine проверяет, принадлежит ли URL этому источнику.
	IsMine(url string) (isMine bool)

	// GetVersions получает список доступных версий.
	// source - источник пакета (URL или путь)
	// Для file:// возвращает ошибку.
	GetVersions(ctx context.Context, source string) (versions []string, err error)

	// ManifestURL формирует URL манифеста для указанной версии.
	// source - источник пакета (URL или путь)
	// version - версия пакета (если "latest" или пустая, получает последнюю версию)
	ManifestURL(ctx context.Context, source string, version string) (manifestURL string, err error)

	// FileURL формирует URL для загрузки файла (если указан без источника).
	// source - источник пакета (URL или путь)
	// version - версия пакета
	// filename - имя файла
	// Если файл уже имеет полный URL, возвращает его как есть.
	FileURL(source string, version string, filename string) (fileURL string, err error)
}

func (u *URI) GetVersions(ctx context.Context) (versions []string, err error) {

	var d dist
	if d, err = u.getDist(); err != nil {
		return
	}

	return d.GetVersions(ctx, u.source)
}

// ManifestURL формирует URL манифеста через выбранный dist.
func (u *URI) ManifestURL(ctx context.Context, verStr string) (manifestURL string, err error) {

	var d dist
	if d, err = u.getDist(); err != nil {
		return
	}

	// Если verStr не указана, проверяем версию из URI
	if verStr == "" {
		if u.version.Original != "" {
			// Если в URI есть версия, используем её
			verStr = u.version.Original
		} else {
			// Если версии нет нигде, передаем "latest" чтобы получить последнюю версию
			verStr = version.LatestVersion
		}
	}

	return d.ManifestURL(ctx, u.source, verStr)
}

// FileURL формирует URL для загрузки файла через выбранный dist.
func (u *URI) FileURL(version string, filename string) (fileURL string, err error) {

	if version == "" {
		version = u.version.Original
	}

	var d dist
	if d, err = u.getDist(); err != nil {
		return
	}

	return d.FileURL(u.source, version, filename)
}

// getDist выбирает подходящий dist для URI.
func (u *URI) getDist() (d dist, err error) {

	// Используем кэш, если dist уже выбран
	if u.dist != nil {
		return u.dist, nil
	}

	// Ищем подходящий dist
	for _, d = range u.dists {
		if d.IsMine(u.source) {
			u.dist = d
			return
		}
	}

	err = errors.New(i18n.Msg("unknown source format"))
	return
}
