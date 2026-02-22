// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"context"
	"errors"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

type dist interface {
	Name() (name string)
	IsMine(url string) (isMine bool)
	GetVersions(ctx context.Context, source string) (versions []string, err error)
	// ManifestURL: при пустой verStr используется версия из URI, иначе "latest".
	ManifestURL(ctx context.Context, source string, version string) (manifestURL string, err error)
	// FileURL: если filename уже полный URL, возвращается как есть.
	FileURL(source string, version string, filename string) (fileURL string, err error)
}

func (u *URI) GetVersions(ctx context.Context) (versions []string, err error) {

	var d dist
	if d, err = u.getDist(); err != nil {
		return
	}

	return d.GetVersions(ctx, u.source)
}

func (u *URI) ManifestURL(ctx context.Context, verStr string) (manifestURL string, err error) {

	var d dist
	if d, err = u.getDist(); err != nil {
		return
	}

	// verStr: при пустой — версия из URI, иначе "latest".
	if verStr == "" {
		if u.version.Original != "" {
			verStr = u.version.Original
		} else {
			verStr = version.LatestVersion
		}
	}

	return d.ManifestURL(ctx, u.source, verStr)
}

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

func (u *URI) getDist() (d dist, err error) {

	if u.dist != nil {
		return u.dist, nil
	}

	for _, d = range u.dists {
		if d.IsMine(u.source) {
			u.dist = d
			return
		}
	}

	return nil, errors.New(i18n.Msg("unknown source format"))
}
