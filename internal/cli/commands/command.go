// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"github.com/seniorGolang/tg/v3/internal/cli/types"
)

// Option - алиас для types.Option
type Option = types.Option

// GlobalOptions - алиас для types.GlobalOptions
type GlobalOptions = types.GlobalOptions

// CommandContext - алиас для types.CommandContext
type CommandContext = types.CommandContext

// Command представляет единый интерфейс для всех команд (встроенных и из плагинов)
type Command interface {
	// GetPath возвращает путь команды в дереве.
	// Алиасы встроены в элементы пути через двоеточие (например, ["plugin:p", "init:i"]).
	// Если алиаса нет, элемент пути без двоеточия (например, ["plugin", "init"]).
	GetPath() (path []string)

	// GetDescription возвращает описание команды
	GetDescription() (description string)

	// GetOptions возвращает опции команды
	GetOptions() (options []Option)

	// Execute выполняет команду
	Execute(ctx CommandContext) (err error)
}
