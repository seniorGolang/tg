// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/executor"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"

	"github.com/spf13/cobra"
)

const (
	requiredArgPrefix = "<"
	requiredArgSuffix = ">"
	optionalArgPrefix = "["
	optionalArgSuffix = "]"
)

func buildCobraCommands(rootCmd *cobra.Command, tree *CommandTree, rootDir string, planner *executor.Planner) (err error) {

	var buildNode func(parent *cobra.Command, node *CommandNode, nodePath []string) (err error)
	buildNode = func(parent *cobra.Command, node *CommandNode, nodePath []string) (err error) {

		if node == nil {
			return
		}

		var cobraCmd *cobra.Command
		if node.command != nil {
			if cobraCmd, err = buildCobraCommand(node.command, rootDir, parent, planner); err != nil {
				return
			}
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
			if err = buildNode(cobraCmd, childNode, append(nodePath, childName)); err != nil {
				return
			}
		}

		if parent != nil {
			parent.AddCommand(cobraCmd)
		} else {
			rootCmd.AddCommand(cobraCmd)
		}

		return
	}

	for name, node := range tree.root {
		if node.alias != "" && name == node.alias {
			continue
		}
		if err = buildNode(nil, node, []string{name}); err != nil {
			return
		}
	}

	return
}

func buildCobraCommand(cmd Command, rootDir string, parent *cobra.Command, planner *executor.Planner) (cobraCmd *cobra.Command, err error) {

	path := cmd.GetPath()
	if len(path) == 0 {
		return nil, nil
	}

	lastElem := path[len(path)-1]

	cmdName, alias := parsePathElement(lastElem)

	var options []Option
	if planner != nil {
		if pluginCmd, ok := cmd.(*lazyPluginCommand); ok {
			var mergedOpts []models.OptionInfo
			if mergedOpts, err = planner.GetMergedOptionsForCommand(pluginCmd.metadata.pluginName, pluginCmd.metadata.command.Path); err != nil {
				return nil, err
			}
			options = convertOptionInfoToOptions(mergedOpts)
		} else {
			options = cmd.GetOptions()
		}
	} else {
		options = cmd.GetOptions()
	}

	positionalOptions := getPositionalOptions(options)
	useStr := cmdName
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
		Run:   createCommandRunner(cmd, rootDir, planner),
	}

	if len(positionalOptions) > 0 {
		requiredCount := 0
		for _, opt := range positionalOptions {
			if opt.Required {
				requiredCount++
			}
		}
		if requiredCount > 0 {
			cobraCmd.Args = func(c *cobra.Command, args []string) (err error) {

				rootCmd := c.Root()
				failOnMissing, _ := rootCmd.PersistentFlags().GetBool(GlobalFlagFailOnMissing)

				if failOnMissing && len(args) < requiredCount {
					return fmt.Errorf(i18n.Msg("requires at least %d arg(s), only received %d"), requiredCount, len(args))
				}
				return
			}
		}
	}

	if alias != "" {
		cobraCmd.Aliases = []string{alias}
	}

	for _, opt := range options {
		addFlagFromOption(cobraCmd, opt)
	}

	return
}

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
			if err = selectedCobraCmd.RunE(selectedCobraCmd, []string{}); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
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
