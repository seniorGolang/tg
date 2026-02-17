// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

type pluginLoader interface {
	GetInfo(packageName string) (installation *models.Installation, err error)
	LoadExecutor(name string, rootDir string) (wasmHost *host.Host, err error)
	GetList() (plugins []models.Installation, err error)
}
