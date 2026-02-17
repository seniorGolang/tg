// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

// convertCommandsToModel конвертирует plugin.Command в CommandInfo для БД.
func convertCommandsToModel(commands []plugin.Command) (commandInfos []models.CommandInfo) {

	commandInfos = make([]models.CommandInfo, 0, len(commands))
	for _, cmd := range commands {
		commandInfos = append(commandInfos, models.CommandInfo{
			Path:        cmd.Path,
			Description: cmd.Description,
			Options:     convertOptionsToModel(cmd.Options),
		})
	}

	return
}

// convertOptionsToModel конвертирует plugin.Option в OptionInfo для БД.
func convertOptionsToModel(options []plugin.Option) (optionInfos []models.OptionInfo) {

	optionInfos = make([]models.OptionInfo, 0, len(options))
	for _, opt := range options {
		optionInfos = append(optionInfos, models.OptionInfo{
			Name:         opt.Name,
			Short:        opt.Short,
			Type:         opt.Type,
			Description:  opt.Description,
			Required:     opt.Required,
			Default:      opt.Default,
			IsPositional: opt.IsPositional,
		})
	}

	return
}
