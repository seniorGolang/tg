// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package storage

import (
	"os"
)

func EnsureDir(path string) (err error) {

	return os.MkdirAll(path, FilePermDir)
}
