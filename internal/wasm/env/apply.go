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

// Apply применяет переменные окружения к конфигурации WASM модуля.
// Всегда автоматически добавляет переменную TG_LANG со значением языка в формате ISO2a,
// определённого через i18n (i18n.GetLanguage() уже учитывает TG_LANG если она установлена).
func Apply(cfg wazero.ModuleConfig, allowedEnvVars []string) (result wazero.ModuleConfig) {

	envVars := resolve(allowedEnvVars)

	for key, value := range envVars {
		cfg = cfg.WithEnv(key, value)
	}

	// Всегда добавляем TG_LANG со значением, определённым через i18n
	// i18n.GetLanguage() уже проверяет TG_LANG первым делом, если она установлена
	tgLang := i18n.GetLanguage()
	cfg = cfg.WithEnv(envVarTGLang, tgLang)

	result = cfg
	return
}
