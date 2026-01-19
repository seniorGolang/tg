// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

type docItem struct {
	name         string
	version      string
	description  string
	doc          string
	installation *models.Installation
}
