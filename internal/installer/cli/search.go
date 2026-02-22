// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func (inst *Installer) HandleSearch(ctx context.Context, args []string) (err error) {

	if len(args) == 0 {
		return errors.New(i18n.Msg("Search query not specified"))
	}

	query := args[0]
	var packages []models.Package
	if packages, err = inst.manifestManager.SearchPackages(ctx, query); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to perform search: %w"), err)
	}

	for _, pkg := range packages {
		fmt.Printf(i18n.Msg("%s - %s")+"\n", pkg.Name, pkg.Descr)
	}

	return
}
