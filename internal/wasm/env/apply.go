// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package env

import (
	"github.com/tetratelabs/wazero"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	envVarTGLang = "TG_LANG"
)

func Apply(cfg wazero.ModuleConfig, allowedEnvVars []string) (result wazero.ModuleConfig) {

	envVars := ResolveEnvVars(allowedEnvVars)

	result = cfg
	for key, value := range envVars {
		result = result.WithEnv(key, value)
	}

	return result.WithEnv(envVarTGLang, i18n.GetLanguage())
}
