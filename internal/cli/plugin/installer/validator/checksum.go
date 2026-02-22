// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func ValidateChecksum(pluginTGP []byte, pluginSHA256 []byte) (err error) {

	hash := sha256.Sum256(pluginTGP)
	computedHash := hex.EncodeToString(hash[:])
	expectedHash := strings.TrimSpace(string(pluginSHA256))

	if computedHash != expectedHash {
		err = fmt.Errorf(i18n.Msg("Checksum mismatch:\n  computed:   %s\n  expected:  %s"), computedHash, expectedHash)
		return
	}

	return
}
