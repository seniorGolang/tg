// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type pathElement struct {
	name  string
	alias string
}

type CommandNode struct {
	name     string
	alias    string
	command  Command
	children map[string]*CommandNode
}

type CommandTree struct {
	root map[string]*CommandNode
}

func NewCommandTree() (tree *CommandTree) {
	return &CommandTree{
		root: make(map[string]*CommandNode),
	}
}

// parsePathElement: "plugin:p" -> name="plugin", alias="p"; "plugin" -> name="plugin", alias=""
func parsePathElement(elem string) (name string, alias string) {

	parts := strings.Split(elem, pathAliasSeparator)
	if len(parts) == 2 {
		name = parts[0]
		alias = parts[1]
		return
	}
	return parts[0], ""
}

func (t *CommandTree) RegisterCommand(cmd Command) (err error) {

	path := cmd.GetPath()
	if len(path) == 0 {
		err = errors.New(i18n.Msg("Command path cannot be empty"))
		return
	}

	pathElements := make([]pathElement, 0, len(path))
	for _, elem := range path {
		name, alias := parsePathElement(elem)
		pathElements = append(pathElements, pathElement{name: name, alias: alias})
	}
	return t.registerCommandWithAliases(pathElements, cmd)
}

func (t *CommandTree) registerCommandWithAliases(path []pathElement, cmd Command) (err error) {

	currentLevel := t.root

	for level, elem := range path {
		if existingNode, exists := currentLevel[elem.name]; exists {
			if level == len(path)-1 {
				if existingNode.command != nil {
					existingIsBuiltin := isBuiltinCommand(existingNode.command)
					newIsBuiltin := isBuiltinCommand(cmd)

					existingPath := existingNode.command.GetPath()
					newPath := buildPathFromElements(path)

					// Встроенные команды имеют приоритет над плагинными при конфликте путей.
					switch {
					case existingIsBuiltin && !newIsBuiltin:
						slog.Warn(i18n.Msg("Command conflict detected"),
							"existing", strings.Join(existingPath, " "),
							"existing_type", i18n.Msg("builtin"),
							"new", strings.Join(newPath, " "),
							"new_type", i18n.Msg("plugin"),
							"action", i18n.Msg("Builtin command has priority, plugin command will not be registered"))
						return
					case !existingIsBuiltin && newIsBuiltin:
						slog.Warn(i18n.Msg("Command conflict detected"),
							"existing", strings.Join(existingPath, " "),
							"existing_type", i18n.Msg("plugin"),
							"new", strings.Join(newPath, " "),
							"new_type", i18n.Msg("builtin"),
							"action", i18n.Msg("Builtin command has priority, replacing plugin command"))
						existingNode.command = cmd
						return
					default:
						cmdType := i18n.Msg("builtin")
						if !existingIsBuiltin {
							cmdType = i18n.Msg("plugin")
						}
						slog.Warn(i18n.Msg("Command conflict detected"),
							"existing", strings.Join(existingPath, " "),
							"new", strings.Join(newPath, " "),
							"type", cmdType,
							"action", i18n.Msg("First command has priority"))
						return
					}
				}
				existingNode.command = cmd
				return
			}
			currentLevel = existingNode.children
		} else {
			newNode := &CommandNode{
				name:     elem.name,
				alias:    elem.alias,
				children: make(map[string]*CommandNode),
			}

			if level == len(path)-1 {
				newNode.command = cmd
			}

			currentLevel[elem.name] = newNode

			if elem.alias != "" {
				if conflictCmd := t.findAliasConflictAtLevel(elem.alias, currentLevel); conflictCmd != nil {
					existingPath := conflictCmd.GetPath()
					newPath := buildPathFromElements(path)
					slog.Warn(i18n.Msg("Alias conflict detected"),
						"alias", elem.alias,
						"existing", strings.Join(existingPath, " "),
						"new", strings.Join(newPath, " "),
						"action", i18n.Msg("Alias will not be created"))
				} else {
					currentLevel[elem.alias] = newNode
				}
			}

			currentLevel = newNode.children
		}
	}

	return
}

func (t *CommandTree) findAliasConflictAtLevel(alias string, level map[string]*CommandNode) (conflictCmd Command) {

	if node, exists := level[alias]; exists && node.command != nil {
		return node.command
	}
	return
}

func buildPathFromElements(elements []pathElement) (path []string) {

	path = make([]string, 0, len(elements))
	for _, elem := range elements {
		if elem.alias != "" {
			path = append(path, fmt.Sprintf("%s%s%s", elem.name, pathAliasSeparator, elem.alias))
		} else {
			path = append(path, elem.name)
		}
	}
	return
}

func (t *CommandTree) FindCommand(path []string) (cmd Command, err error) {

	currentLevel := t.root

	for i, elemName := range path {
		name, _ := parsePathElement(elemName)

		node, exists := currentLevel[name]
		if !exists {
			node, exists = currentLevel[elemName]
			if !exists {
				return nil, fmt.Errorf(i18n.Msg("Command not found: %s"), strings.Join(path, " "))
			}
		}

		if i == len(path)-1 {
			if node.command != nil {
				cmd = node.command
				return
			}
			return nil, fmt.Errorf(i18n.Msg("Command not found: %s"), strings.Join(path, " "))
		}

		currentLevel = node.children
	}

	return nil, fmt.Errorf(i18n.Msg("Command not found: %s"), strings.Join(path, " "))
}

func (n *CommandNode) GetSubcommands() (subcommands []Command) {

	subcommands = make([]Command, 0)
	for childName, child := range n.children {
		if childName != child.name {
			continue
		}
		if child.command != nil {
			subcommands = append(subcommands, child.command)
		}
	}
	return
}

func (n *CommandNode) GetChildNodes() (childNodes []*CommandNode) {

	childNodes = make([]*CommandNode, 0)
	for childName, child := range n.children {
		if childName != child.name {
			continue
		}
		childNodes = append(childNodes, child)
	}
	return
}
