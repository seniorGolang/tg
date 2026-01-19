// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package i18n

import (
	"embed"
	"fmt"

	"github.com/goccy/go-json"
)

//go:embed lang/*.json
var translationsFS embed.FS

var translations map[string]string

func init() {
	translations = make(map[string]string)

	lang := detectLanguage()
	if lang == langCodeEN {
		return
	}

	langPath := fmt.Sprintf("%s%s.json", langDir, lang)
	var langData []byte
	var err error
	langData, err = translationsFS.ReadFile(langPath)
	if err != nil {
		return
	}

	var langTranslations map[string]string
	if err = json.Unmarshal(langData, &langTranslations); err != nil {
		return
	}

	for key, value := range langTranslations {
		translations[key] = value
	}
}
