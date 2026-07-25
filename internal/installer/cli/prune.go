// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// HandlePrune оставляет только последнюю установленную версию пакетов, ничего не загружая.
func (inst *Installer) HandlePrune(ctx context.Context, args []string) (err error) {

	var installations []models.Installation
	if installations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get list of installations: %w"), err)
	}

	groups := groupInstallations(installations)
	keys := selectGroups(groups, args)
	if len(keys) == 0 {
		return errors.New(i18n.Msg("No packages to prune"))
	}

	prunedTotal := 0
	for _, key := range keys {
		group := groups[key]
		if len(group) < 2 {
			continue
		}

		keep := latestVersion(group)
		if keep.Original == "" {
			continue
		}

		var removed []string
		if removed, err = inst.keepOnlyVersion(ctx, group, keep); err != nil {
			return
		}
		if len(removed) == 0 {
			continue
		}

		// Published skills адресуются по имени, поэтому снятие старых версий стирает общий путь: переустанавливаем skills выжившей версии.
		if err = inst.installationManager.InstallSkills(ctx, []string{key.name}); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to install skills: %w"), err)
		}

		prunedTotal += len(removed)
		pterm.Success.Printf(i18n.Msg("Package %s: kept version %s, removed: %s")+"\n", key.name, keep.Original, strings.Join(removed, ", "))
	}

	if prunedTotal == 0 {
		pterm.Info.Println(i18n.Msg("No extra package versions found"))
	}

	return
}
