// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"net/url"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/uri/file"
	"github.com/seniorGolang/tg/v3/internal/installer/uri/github"
	"github.com/seniorGolang/tg/v3/internal/installer/uri/proxy"
)

type URI struct {
	source      string
	packageName string

	version models.Version

	parsedURL *url.URL

	dist  dist
	dists []dist
}

// New: поддерживаемые форматы (требуется URL со схемой):
//
// URL:packageName@version:
//   - http://example.com/path:myapp@1.0.0
//   - https://example.com:8080/path:myapp@v1.2.3
//   - https://example.com/path:myapp@>=1.0.0
//
// URL@version:
//   - http://example.com/path@1.0.0
//   - https://example.com/path@v1.2.3
//   - https://github.com/owner/repo@^1.0.0
//
// Примечание: Форматы packageName@version и source/packageName@version должны быть
// нормализованы в полный формат URL:packageName@version перед вызовом New.
func New(spec string, opts ...Option) (u URI, err error) {

	for _, opt := range opts {
		if opt != nil {
			opt(&u)
		}
	}

	if len(u.dists) == 0 {
		u.dists = []dist{
			file.NewDist(),
			proxy.NewDist(),
			github.NewDist(),
		}
	}

	if err = u.parse(spec); err != nil {
		return
	}

	return
}
