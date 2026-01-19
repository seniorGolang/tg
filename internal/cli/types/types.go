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
	Short        string
	Type         string
	Description  string
	Required     bool
	Default      any
	IsPositional bool // true для позиционных аргументов (не флагов)
}

// GlobalOptions содержит глобальные опции, доступные для всех команд
type GlobalOptions struct {
	LogLevel      string // Уровень логирования: "debug", "info", "warn", "error" (по умолчанию: "info")
	HideCmd       bool   // Скрывать вывод собранной команды после интерактивного режима (по умолчанию: false)
	FailOnMissing bool   // Выводить ошибку вместо интерактивного режима (по умолчанию: false)
	Scope         string // Scope для выполнения команды (по умолчанию: пустая строка, используется текущий scope)
}

// CommandContext содержит контекст выполнения команды
type CommandContext struct {
	Context      context.Context // Контекст для отмены выполнения
	RootDir      string
	Options      map[string]any
	Args         []string // Позиционные аргументы команды
	CommandPath  []string
	Logger       plugin.Logger
	StateManager *state.Manager
	GlobalOpts   GlobalOptions // Глобальные опции для всех команд
}
