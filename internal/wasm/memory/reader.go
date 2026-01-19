// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func ReadString(h *host.Host, ptr uint32, length uint32) (str string, err error) {

	if length == 0 {
		return "", nil
	}

	var data []byte
	if data, err = Read(h, ptr, length); err != nil {
		return "", err
	}

	str = string(data)
	return
}
