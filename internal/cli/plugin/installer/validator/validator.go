// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package validator

import (
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func ValidatePlugin(pluginTGP []byte, pluginSHA256 []byte, pluginInfo plugin.Info, expectedName string, expectedVersion string) (err error) {

	if err = ValidateChecksum(pluginTGP, pluginSHA256); err != nil {
		err = fmt.Errorf(i18n.Msg("Checksum validation failed: %w"), err)
		return
	}

	if err = ValidateMetadata(pluginInfo, expectedName, expectedVersion); err != nil {
		err = fmt.Errorf(i18n.Msg("Metadata validation failed: %w"), err)
		return
	}

	if err = ValidateWASM(pluginTGP, pluginInfo); err != nil {
		err = fmt.Errorf(i18n.Msg("WASM structure validation failed: %w"), err)
		return
	}

	return
}
