// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"strings"

	"github.com/spf13/cobra"
)

// getCommandPath исключает корневую "tg" и плейсхолдеры позиционных аргументов.
func getCommandPath(cmd *cobra.Command) (commandPath []string) {

	commandPath = make([]string, 0)
	current := cmd
	for current != nil && current.Use != "" {
		// Пропускаем корневую команду
		if current.Use != cmdNameTG {
			// Извлекаем только имя команды, убирая плейсхолдеры позиционных аргументов (<arg>, [arg])
			cmdName := extractCommandName(current.Use)
			commandPath = append([]string{cmdName}, commandPath...)
		}
		current = current.Parent()
	}
	return
}

// extractCommandName извлекает имя команды из Use строки, убирая плейсхолдеры
// "install <package>" -> "install"
// "init [name]" -> "init"
func extractCommandName(useStr string) (cmdName string) {

	// Разбиваем по пробелам и берём только первую часть (имя команды)
	parts := strings.Fields(useStr)
	if len(parts) > 0 {
		cmdName = parts[0]
		return
	}
	cmdName = useStr
	return
}

// addFlagFromOption добавляет флаг в команду на основе опции
func addFlagFromOption(cmd *cobra.Command, opt Option) {

	// Пропускаем позиционные опции - они обрабатываются отдельно
	if opt.IsPositional {
		return
	}
	switch opt.Type {
	case optionTypeString:
		defaultVal := ""
		if opt.Default != nil {
			if str, ok := opt.Default.(string); ok {
				defaultVal = str
			}
		}
		if opt.Short != "" {
			cmd.Flags().StringP(opt.Name, opt.Short, defaultVal, opt.Description)
		} else {
			cmd.Flags().String(opt.Name, defaultVal, opt.Description)
		}
	case optionTypeInt:
		defaultVal := 0
		if opt.Default != nil {
			if i, ok := opt.Default.(int); ok {
				defaultVal = i
			} else if f, ok := opt.Default.(float64); ok {
				defaultVal = int(f)
			}
		}
		if opt.Short != "" {
			cmd.Flags().IntP(opt.Name, opt.Short, defaultVal, opt.Description)
		} else {
			cmd.Flags().Int(opt.Name, defaultVal, opt.Description)
		}
	case optionTypeBool:
		defaultVal := false
		if opt.Default != nil {
			if b, ok := opt.Default.(bool); ok {
				defaultVal = b
			}
		}
		if opt.Short != "" {
			cmd.Flags().BoolP(opt.Name, opt.Short, defaultVal, opt.Description)
		} else {
			cmd.Flags().Bool(opt.Name, defaultVal, opt.Description)
		}
	}
}
