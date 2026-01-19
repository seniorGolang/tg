// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// PluginParams содержит параметры для создания плагина.
type PluginParams struct {
	Name       string
	Command    string
	DeployType string
	License    string
	ModuleName string
	Kind       string
}

// Validator валидирует и нормализует параметры плагина.
type Validator struct{}

// ValidateAndNormalize валидирует и нормализует параметры плагина.
func (v *Validator) ValidateAndNormalize(params *PluginParams) (err error) {

	// Проверка обязательных полей
	if params.Name == "" {
		err = errors.New(i18n.Msg("Plugin name is required"))
		return
	}
	// Command не обязателен для плагинов-трансформеров

	// Валидация имени плагина
	if err = isValidPluginName(params.Name); err != nil {
		err = fmt.Errorf(i18n.Msg("Invalid plugin name: %w"), err)
		return
	}

	// Нормализация имени плагина
	params.Name = normalizePluginName(params.Name)

	// Установка значений по умолчанию
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
