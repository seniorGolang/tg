// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"errors"
	"log/slog"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/generator"
	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func HandlePluginAdd(ctx types.CommandContext) (err error) {

	name, _ := ctx.Options[optionKeyName].(string)
	command, _ := ctx.Options[optionKeyCommand].(string)
	dir, _ := ctx.Options[optionKeyDir].(string)
	license, _ := ctx.Options[optionKeyLicense].(string)
	moduleName, _ := ctx.Options[optionKeyModuleName].(string)
	kind, _ := ctx.Options[optionKeyKind].(string)

	logger := slog.With(
		slog.String("operation", "add_plugin"),
		slog.Group("plugin",
			slog.String("name", name),
			slog.String("command", command),
			slog.String("dir", dir),
		),
	)
	logger.Info(i18n.Msg("Adding plugin"))
	if err = generator.RunAdd(ctx.RootDir, name, command, dir, license, moduleName, kind); err != nil {
		logger.Error(i18n.Msg("Error adding plugin"), slog.String("error", err.Error()))
		return errors.New(i18n.Msg("Error adding plugin") + ": " + err.Error())
	}
	logger.Info(i18n.Msg("Plugin successfully added"))
	return
}
