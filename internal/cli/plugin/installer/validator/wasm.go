// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package validator

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

const (
	wasmMinSize       = 8
	wasmMagicSize     = 4
	wasmVersionSize   = 4
	wasmVersionOffset = 4
)

var (
	wasmMagic = []byte{0x00, 0x61, 0x73, 0x6D}
)

func ValidateWASM(pluginTGP []byte, pluginInfo plugin.Info) (err error) {

	if len(pluginTGP) < wasmMinSize {
		err = errors.New(i18n.Msg("Plugin file too small for WASM module"))
		return
	}

	if len(pluginTGP) < wasmMagicSize {
		err = errors.New(i18n.Msg("Plugin file too small"))
		return
	}

	for i := 0; i < wasmMagicSize; i++ {
		if pluginTGP[i] != wasmMagic[i] {
			err = fmt.Errorf(i18n.Msg("Invalid WASM magic header: expected \\0asm, got %v"), pluginTGP[:wasmMagicSize])
			return
		}
	}

	if len(pluginTGP) < wasmMinSize {
		err = errors.New(i18n.Msg("Plugin file too small to check WASM version"))
		return
	}

	wasmVersion := pluginTGP[wasmVersionOffset : wasmVersionOffset+wasmVersionSize]

	if wasmVersion[0] == 0x00 && wasmVersion[1] == 0x00 && wasmVersion[2] == 0x00 && wasmVersion[3] == 0x00 {
		err = errors.New(i18n.Msg("Unsupported WASM version: 0"))
		return
	}

	if pluginInfo.Name == "" {
		err = errors.New(i18n.Msg("Failed to get plugin info from WASM module"))
		return
	}

	return
}
