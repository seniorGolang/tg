// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

type docItem struct {
	doc          string
	name         string
	version      string
	description  string
	installation *models.Installation
}
