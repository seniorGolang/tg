// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// HandleUpgrade обрабатывает команду upgrade.
func (inst *Installer) HandleUpgrade(ctx context.Context, args []string) (err error) {

	return errors.New(i18n.Msg("Upgrade command not implemented"))
}
