// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"github.com/tetratelabs/wazero/api"
)

// Host: пакет memory не импортирует host, чтобы избежать циклических зависимостей.
type Host interface {
	GetModule() (module api.Module)
	GetMalloc() (malloc api.Function)
	GetFree() (free api.Function)
}
