// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type PluginParams struct {
	Name       string
	Kind       string
	License    string
	Command    string
	DeployType string
	ModuleName string
}

type Validator struct{}

func (v *Validator) ValidateAndNormalize(params *PluginParams) (err error) {

	if params.Name == "" {
		return errors.New(i18n.Msg("Plugin name is required"))
	}

	if err = isValidPluginName(params.Name); err != nil {
		return fmt.Errorf(i18n.Msg("Invalid plugin name: %w"), err)
	}

	params.Name = normalizePluginName(params.Name)

	if params.License == "" {
		params.License = DefaultLicense
	}
	if params.ModuleName == "" {
		params.ModuleName = DefaultModuleName
	}
	if params.DeployType == "" {
		params.DeployType = DeployTypeNone
	}

	return
}
