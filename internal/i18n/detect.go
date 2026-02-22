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
		if tgLang := strings.ToLower(strings.TrimSpace(os.Getenv(envTGLang))); tgLang != "" {
			if len(tgLang) == 2 {
				detectedLanguage = tgLang
				return
			}
		}

		// LC_ALL, LC_MESSAGES, LANG — в порядке приоритета.
		envVars := []string{envLCAll, envLCMessages, envLANG}

		for _, envVar := range envVars {
			locale := os.Getenv(envVar)
			if locale == "" {
				continue
			}

			locale = strings.ToLower(locale)

			if dotIndex := strings.Index(locale, "."); dotIndex != -1 {
				locale = locale[:dotIndex]
			}

			parts := strings.FieldsFunc(locale, func(r rune) bool {
				return r == '_' || r == '-'
			})

			if len(parts) > 0 && len(parts[0]) == 2 {
				detectedLanguage = parts[0]
				return
			}
		}

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
