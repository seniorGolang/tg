// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"github.com/seniorGolang/tg/v3/internal/cli/types"
)

type Option = types.Option
type GlobalOptions = types.GlobalOptions
type CommandContext = types.CommandContext

// GetPath: алиасы встроены в элементы пути через двоеточие (например, ["plugin:p", "init:i"]).
type Command interface {
	GetPath() (path []string)
	GetDescription() (description string)

	GetOptions() (options []Option)

	Execute(ctx CommandContext) (err error)
}
