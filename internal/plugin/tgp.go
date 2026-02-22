// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package plugin

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	gzipMagicLen = 2
)

var (
	gzipMagic = []byte{0x1f, 0x8b}
)

// DecodeTGPBytes: при магии gzip (0x1f 0x8b) в начале raw распаковывает данные; иначе возвращает raw без изменений.
func DecodeTGPBytes(raw []byte) (wasmBytes []byte, err error) {

	if len(raw) < gzipMagicLen {
		return raw, nil
	}

	if !bytes.Equal(raw[:gzipMagicLen], gzipMagic) {
		return raw, nil
	}

	var gzReader *gzip.Reader
	if gzReader, err = gzip.NewReader(bytes.NewReader(raw)); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to create gzip reader for .tgp: %w"), err)
		return
	}
	defer func() { _ = gzReader.Close() }()

	if wasmBytes, err = io.ReadAll(gzReader); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to decompress .tgp: %w"), err)
		return
	}

	return
}
