// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package types

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/state"
)

// Option описывает опцию команды
type Option struct {
	Name         string
	Type         string
	Short        string
	Default      any
	Required     bool
	Description  string
	IsPositional bool // true для позиционных аргументов (не флагов)
}

// GlobalOptions содержит глобальные опции, доступные для всех команд
type GlobalOptions struct {
	Scope         string // Scope для выполнения команды (по умолчанию: пустая строка, используется текущий scope)
	LogLevel      string // Уровень логирования: "debug", "info", "warn", "error" (по умолчанию: "info")
	HideCmd       bool   // Скрывать вывод собранной команды после интерактивного режима (по умолчанию: false)
	FailOnMissing bool   // Выводить ошибку вместо интерактивного режима (по умолчанию: false)
}

// CommandContext содержит контекст выполнения команды
type CommandContext struct {
	Args         []string // Позиционные аргументы команды
	Logger       plugin.Logger
	Context      context.Context
	RootDir      string
	Options      map[string]any
	GlobalOpts   GlobalOptions
	CommandPath  []string
	StateManager *state.Manager
}
