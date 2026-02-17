// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type PluginCreator struct{}

func (c *PluginCreator) Create(pluginDir string, data TemplateData) (err error) {

	if err = os.MkdirAll(pluginDir, 0755); err != nil {
		return
	}

	// В plugins/{name}/ создаём код плагина, tgp.go, wasm.go и manifest.go
	// plugin.json генерируется при сборке через go generate
	files := map[string]string{
		"plugin.go":   "templates/plugin_plugin.go.tmpl",
		"wasm.go":     "templates/plugin_wasm.go.tmpl",
		"tgp.go":      "templates/plugin_tgp.go.tmpl",
		"manifest.go": "templates/plugin_manifest.go.tmpl",
		"plugin.md":   "templates/plugin_plugin.md.tmpl",
	}

	for filename, templatePath := range files {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(pluginDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), filename, err)
		}
	}

	i18nPluginsDir := filepath.Join("i18n", "plugins", data.PluginNameOriginal)
	if err = os.MkdirAll(i18nPluginsDir, 0755); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/plugins directory", err)
		return
	}

	pluginRuFile := filepath.Join(i18nPluginsDir, "ru.json")
	var translation string
	if data.Description == "This is an plugin" {
		translation = "Это тестовый плагин"
	} else {
		translation = data.Description
	}
	pluginRuContent := fmt.Sprintf(`{
  "%s": "%s"
}
`, data.Description, translation)
	if err = os.WriteFile(pluginRuFile, []byte(pluginRuContent), 0600); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/plugins/"+data.PluginNameOriginal+"/ru.json", err)
		return
	}

	return
}

func (c *PluginCreator) Exists(pluginDir string) (exists bool) {

	var err error
	if _, err = os.Stat(pluginDir); err == nil {
		exists = true
		return
	}

	return
}
