// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"errors"
	"os"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func EnsureDir(path string) (err error) {

	if path == "" {
		return errors.New(i18n.Msg("path cannot be empty"))
	}
	return os.MkdirAll(path, FilePermDir)
}
