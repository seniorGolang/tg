// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"fmt"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func PromptCommandArgs(cmd Command, cobraCmd *cobra.Command, commandOptions []Option, commandPath []string) (args []string) {

	positionalOptions := getPositionalOptions(commandOptions)
	if len(positionalOptions) == 0 {
		args = nil
		return
	}

	args = make([]string, 0, len(positionalOptions))

	i := 0
	for i < len(positionalOptions) {
		opt := positionalOptions[i]

		promptText := opt.Description
		if opt.Required {
			promptText = fmt.Sprintf("%s%s", promptText, i18n.Msg(" (required)"))
		}

		arg, _ := pterm.DefaultInteractiveTextInput.
			Show(promptText)

		if arg == "" {
			if opt.Required {
				pterm.Warning.Println(i18n.Msg("This field is required"))
				continue
			}
			break
		}

		args = append(args, arg)

		if cobraCmd.Args != nil {
			if err := cobraCmd.Args(cobraCmd, args); err != nil {
				if strings.Contains(err.Error(), "requires") || strings.Contains(err.Error(), "accepts") {
					i++
					continue
				}
				pterm.Warning.Printfln(i18n.Msg("Invalid argument: %s"), err)
				args = args[:len(args)-1]
				continue
			}
		}

		hasMoreRequired := false
		for j := i + 1; j < len(positionalOptions); j++ {
			if positionalOptions[j].Required {
				hasMoreRequired = true
				break
			}
		}

		if !hasMoreRequired {
			return
		}

		i++
	}

	return
}
