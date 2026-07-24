// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/skills"
)

func (inst *Installer) HandlePkgSkillsInstall(ctx context.Context, args []string) (err error) {

	if err = inst.installationManager.InstallSkills(ctx, args); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to install skills: %w"), err)
	}
	pterm.Success.Println(i18n.Msg("Skills installed"))
	return
}

func (inst *Installer) HandleHostSkillsInstall(ctx context.Context) (err error) {

	opts := skills.FromContext(ctx)
	opts.Enabled = true
	if err = skills.InstallHost(opts); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to install skills: %w"), err)
	}
	pterm.Success.Println(i18n.Msg("Skills installed"))
	return
}
