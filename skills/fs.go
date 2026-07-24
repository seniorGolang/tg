// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skillfs

import "embed"

//go:embed all:tg all:tg-plugin
var FS embed.FS
