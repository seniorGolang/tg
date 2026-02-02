// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package main

import (
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli"
	"github.com/seniorGolang/tg/v3/internal/cli/commands"
	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func runRoot(cmd *cobra.Command, _ []string) {

	if versionFlag, _ := cmd.PersistentFlags().GetBool("version"); versionFlag {
		versionText := i18n.Msg("Version")
		pterm.Print(pterm.Green(versionText), " ", pterm.Cyan("tg"), " ", cli.Version, "\n")
		return
	}

	commandPathStr := commands.PromptCommandSelection()
	if commandPathStr == "" {
		return
	}

	commandPath := strings.Fields(commandPathStr)
	var selectedCobraCmd *cobra.Command
	var err error
	if selectedCobraCmd, _, err = cmd.Root().Find(commandPath); err != nil {
		slog.Error(i18n.Msg("Failed to find command"), "path", strings.Join(commandPath, " "), "error", err)
		return
	}

	if selectedCobraCmd.Run != nil {
		selectedCobraCmd.Run(selectedCobraCmd, []string{})
	} else if selectedCobraCmd.RunE != nil {
		_ = selectedCobraCmd.RunE(selectedCobraCmd, []string{})
	}
}
