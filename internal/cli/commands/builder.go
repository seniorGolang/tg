// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"

	"github.com/spf13/cobra"
)

const (
	requiredArgPrefix = "<"
	requiredArgSuffix = ">"
	optionalArgPrefix = "["
	optionalArgSuffix = "]"
)

// buildCobraCommands строит cobra команды из дерева команд
func buildCobraCommands(rootCmd *cobra.Command, tree *CommandTree, rootDir string) {

	// Рекурсивно строим команды из дерева
	var buildNode func(parent *cobra.Command, node *CommandNode, nodePath []string)
	buildNode = func(parent *cobra.Command, node *CommandNode, nodePath []string) {
		if node == nil {
			return
		}

		var cobraCmd *cobra.Command
		if node.command != nil {
			cobraCmd = buildCobraCommand(node.command, rootDir, parent)
		} else {
			cobraCmd = &cobra.Command{
				Use:   node.name,
				Short: fmt.Sprintf(i18n.Msg("Command %s"), node.name),
				Run:   createSubcommandSelector(node, rootDir),
			}
		}

		if node.alias != "" {
			cobraCmd.Aliases = []string{node.alias}
		}

		for childName, childNode := range node.children {
			if childNode.alias != "" && childName == childNode.alias {
				continue
			}
			buildNode(cobraCmd, childNode, append(nodePath, childName))
		}

		if parent != nil {
			parent.AddCommand(cobraCmd)
		} else {
			rootCmd.AddCommand(cobraCmd)
		}
	}

	for name, node := range tree.root {
		if node.alias != "" && name == node.alias {
			continue
		}
		buildNode(nil, node, []string{name})
	}
}

func buildCobraCommand(cmd Command, rootDir string, parent *cobra.Command) (cobraCmd *cobra.Command) {

	path := cmd.GetPath()
	if len(path) == 0 {
		cobraCmd = nil
		return
	}

	lastElem := path[len(path)-1]

	cmdName, alias := parsePathElement(lastElem)

	useStr := cmdName
	positionalOptions := getPositionalOptions(cmd.GetOptions())
	if len(positionalOptions) > 0 {
		argParts := make([]string, 0, len(positionalOptions))
		for _, opt := range positionalOptions {
			if opt.Required {
				argParts = append(argParts, fmt.Sprintf("%s%s%s", requiredArgPrefix, opt.Name, requiredArgSuffix))
			} else {
				argParts = append(argParts, fmt.Sprintf("%s%s%s", optionalArgPrefix, opt.Name, optionalArgSuffix))
			}
		}
		useStr = cmdName + " " + strings.Join(argParts, " ")
	}

	cobraCmd = &cobra.Command{
		Use:   useStr,
		Short: cmd.GetDescription(),
		Run:   createCommandRunner(cmd, rootDir),
	}

	if len(positionalOptions) > 0 {
		requiredCount := 0
		for _, opt := range positionalOptions {
			if opt.Required {
				requiredCount++
			}
		}
		if requiredCount > 0 {
			cobraCmd.Args = func(cmd *cobra.Command, args []string) (err error) {
				rootCmd := cmd.Root()
				failOnMissing, _ := rootCmd.PersistentFlags().GetBool(GlobalFlagFailOnMissing)

				if failOnMissing && len(args) < requiredCount {
					err = fmt.Errorf(i18n.Msg("requires at least %d arg(s), only received %d"), requiredCount, len(args))
					return
				}
				return
			}
		}
	}

	if alias != "" {
		cobraCmd.Aliases = []string{alias}
	}

	for _, opt := range cmd.GetOptions() {
		addFlagFromOption(cobraCmd, opt)
	}

	return
}

// createSubcommandSelector используется, когда команда вызывается без подкоманды (например, "tg plugin").
func createSubcommandSelector(node *CommandNode, rootDir string) (selector func(cobraCmd *cobra.Command, args []string)) {
	return func(cobraCmd *cobra.Command, args []string) {
		cmdPath := getCommandPath(cobraCmd)
		subcommands := node.GetSubcommands()
		selected := PromptSubcommandSelection(cmdPath, subcommands)
		if selected == nil {
			return
		}

		selectedPath := selected.GetPath()
		cmdPath = append(cmdPath, selectedPath[len(cmdPath):]...)
		var selectedCobraCmd *cobra.Command
		var err error
		if selectedCobraCmd, _, err = cobraCmd.Root().Find(cmdPath); err != nil {
			slog.Error(i18n.Msg("Failed to find command"), "path", strings.Join(cmdPath, " "), "error", err)
			return
		}

		if selectedCobraCmd.Run != nil {
			selectedCobraCmd.Run(selectedCobraCmd, []string{})
		} else if selectedCobraCmd.RunE != nil {
			if err := selectedCobraCmd.RunE(selectedCobraCmd, []string{}); err != nil {
				var pluginErr *imports.PluginError
				if errors.As(err, &pluginErr) {
					slog.Error(pluginErr.Message)
				} else {
					slog.Error(i18n.Msg("Command execution error"), "error", err)
				}
			}
		}
	}
}
