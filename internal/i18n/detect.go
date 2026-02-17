// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package i18n

import (
	"os"
	"strings"
	"sync"
)

var (
	detectedLanguage   string
	detectLanguageOnce sync.Once
)

func detectLanguage() (lang string) {

	detectLanguageOnce.Do(func() {
		// Сначала проверяем TG_LANG, если она установлена, используем её значение
		if tgLang := strings.ToLower(strings.TrimSpace(os.Getenv(envTGLang))); tgLang != "" {
			// Проверяем, что значение валидное (2 символа)
			if len(tgLang) == 2 {
				detectedLanguage = tgLang
				return
			}
		}

		// Проверяем переменные окружения в порядке приоритета
		// LC_ALL имеет наивысший приоритет, затем LC_MESSAGES, затем LANG
		envVars := []string{envLCAll, envLCMessages, envLANG}

		for _, envVar := range envVars {
			locale := os.Getenv(envVar)
			if locale == "" {
				continue
			}

			locale = strings.ToLower(locale)

			// Убираем кодировку (.utf-8, .UTF-8 и т.д.)
			if dotIndex := strings.Index(locale, "."); dotIndex != -1 {
				locale = locale[:dotIndex]
			}

			// Разбиваем по подчеркиванию или дефису
			parts := strings.FieldsFunc(locale, func(r rune) bool {
				return r == '_' || r == '-'
			})

			if len(parts) > 0 && len(parts[0]) == 2 {
				detectedLanguage = parts[0]
				return
			}
		}

		// Если ни одна переменная окружения не установлена или не содержит валидный код языка,
		// используем английский по умолчанию
		detectedLanguage = langCodeEN
	})

	return detectedLanguage
}

// GetLanguage: приоритет TG_LANG, LC_ALL, LC_MESSAGES, LANG; по умолчанию "en".
func GetLanguage() (lang string) {

	lang = detectLanguage()
	if lang == "" {
		return langCodeEN
	}
	return
}
