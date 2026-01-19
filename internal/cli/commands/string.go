// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	commandStringSeparator = " "
	globalOptLogLevel      = "log-level"
	globalOptHideCmd       = "hide-cmd"
	globalOptFailOnMissing = "fail-on-missing"
	flagPrefix             = "--"
)

// showBuiltCommand показывает собранную команду через лог
func showBuiltCommand(cmd Command, options map[string]any, args []string, commandPath []string) {

	fullCmd := buildCommandString(commandPath, options, args, false)

	// Выводим команду через slog.Info с полем cmd
	slog.Info(i18n.Msg("run"), "cmd", fullCmd)
	fmt.Println()
}

// buildCommandString строит строку команды из пути, опций и позиционных аргументов
func buildCommandString(commandPath []string, options map[string]any, args []string, useAliases bool) (commandStr string) {

	var parts []string
	parts = append(parts, cmdNameTG)

	// Добавляем путь команды (с алиасами или без)
	for _, elem := range commandPath {
		if useAliases {
			name, alias := parsePathElement(elem)
			if alias != "" {
				parts = append(parts, alias)
			} else {
				parts = append(parts, name)
			}
		} else {
			name, _ := parsePathElement(elem)
			parts = append(parts, name)
		}
	}

	// Добавляем позиционные аргументы
	parts = append(parts, args...)

	// Добавляем опции (флаги)
	for key, value := range options {
		if value != nil && value != "" && value != false && value != 0 {
			switch v := value.(type) {
			case bool:
				if v {
					parts = append(parts, fmt.Sprintf("%s%s", flagPrefix, key))
				}
			case string:
				parts = append(parts, fmt.Sprintf("%s%s", flagPrefix, key), v)
			case int:
				parts = append(parts, fmt.Sprintf("%s%s", flagPrefix, key), fmt.Sprintf("%d", v))
			default:
				parts = append(parts, fmt.Sprintf("%s%s", flagPrefix, key), fmt.Sprintf("%v", v))
			}
		}
	}

	commandStr = strings.Join(parts, commandStringSeparator)
	return
}

// buildCommandArgs строит список аргументов команды из опций, позиционных аргументов и глобальных опций.
// Возвращает массив строк без пути команды (только аргументы и флаги).
func buildCommandArgs(options map[string]any, args []string, globalOpts GlobalOptions) (commandArgs []string) {

	commandArgs = make([]string, 0)

	// Добавляем позиционные аргументы
	commandArgs = append(commandArgs, args...)

	// Добавляем опции команды (флаги) без префиксов --
	for key, value := range options {
		if value != nil && value != "" && value != false && value != 0 {
			switch v := value.(type) {
			case bool:
				if v {
					commandArgs = append(commandArgs, key)
				}
			case string:
				commandArgs = append(commandArgs, key, v)
			case int:
				commandArgs = append(commandArgs, key, fmt.Sprintf("%d", v))
			default:
				commandArgs = append(commandArgs, key, fmt.Sprintf("%v", v))
			}
		}
	}

	// Добавляем глобальные опции, если они заданы (не по умолчанию), без префиксов --
	if globalOpts.LogLevel != "" && globalOpts.LogLevel != logLevelInfo {
		commandArgs = append(commandArgs, globalOptLogLevel, globalOpts.LogLevel)
	}
	if globalOpts.HideCmd {
		commandArgs = append(commandArgs, globalOptHideCmd)
	}
	if globalOpts.FailOnMissing {
		commandArgs = append(commandArgs, globalOptFailOnMissing)
	}

	return
}
