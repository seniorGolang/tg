// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func fileSHA256Hex(path string) (hash string, err error) {

	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func checksumLine(hexHash string) (line string) {
	return "sha256:" + hexHash
}

func formatSize(path string) (s string) {

	var err error
	var info os.FileInfo
	if info, err = os.Stat(path); err != nil {
		return "?"
	}

	n := info.Size()
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(n)/float64(div), "KMGTPE"[exp])
}
