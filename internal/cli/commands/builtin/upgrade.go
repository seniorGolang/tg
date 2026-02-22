// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/cli/plugin/generator"
	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/i18n"

	"golang.org/x/mod/modfile"
)

func HandlePluginUpgrade(ctx types.CommandContext) (err error) {

	var moduleName string
	goModPath := filepath.Join(ctx.RootDir, goModFile)
	var data []byte
	if data, err = os.ReadFile(goModPath); err == nil {
		var mf *modfile.File
		if mf, err = modfile.Parse(goModFile, data, nil); err == nil && mf.Module != nil {
			moduleName = mf.Module.Mod.Path
		}
	}
	if moduleName == "" {
		moduleName = generator.DefaultModuleName
	}

	cicdCreator := &generator.CICDCreator{}
	deployType := cicdCreator.DetectDeployType(ctx.RootDir)

	logger := slog.With(
		slog.String("operation", "upgrade_plugin"),
		slog.String("module-name", moduleName),
		slog.String("deploy-type", deployType),
	)
	logger.Info(i18n.Msg("Updating generated files"))
	if err = generator.RunUpgrade(context.Background(), ctx.RootDir, moduleName, deployType); err != nil {
		logger.Error(i18n.Msg("Error updating files"), slog.String("error", err.Error()))
		return errors.New(i18n.Msg("Error updating files") + ": " + err.Error())
	}
	logger.Info(i18n.Msg("Files successfully updated"))
	return
}
