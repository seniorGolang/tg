// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type RootFilesCreator struct{}

func (c *RootFilesCreator) Create(rootDir string, data TemplateData, moduleName string) (err error) {

	if moduleName == "" {
		moduleName = DefaultModuleName
	}

	rootData := TemplateData{
		ModuleName:          moduleName,
		PluginName:          data.PluginName,
		PluginNameTitleCase: data.PluginNameTitleCase,
		PluginNameSnakeCase: data.PluginNameSnakeCase,
		PluginNameOriginal:  data.PluginNameOriginal,
		Description:         data.Description,
		Author:              data.Author,
		License:             data.License,
		Category:            data.Category,
		Command:             data.Command,
		DeployType:          data.DeployType,
	}

	files := map[string]string{
		GoModFileName: "templates/root_gomod.tmpl",
		"readme.md":   "templates/plugin_readme.tmpl",
		".gitignore":  "templates/gitignore.tmpl",
	}

	for filename, templatePath := range files {
		// В WASM файловая система монтируется в корень "/", поэтому используем относительные пути
		filePath := filename

		// Для .gitignore проверяем существование, чтобы не перезаписать существующий
		if filename == ".gitignore" {
			var statErr error
			if _, statErr = os.Stat(filePath); statErr == nil {
				// Файл уже существует, пропускаем
				continue
			}
		}

		var content string
		if content, err = renderTemplate(templatePath, rootData); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), filename, err)
		}
	}

	i18nDir := "i18n"
	if err = os.MkdirAll(i18nDir, 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n directory", err)
	}

	i18nCoreDir := filepath.Join(i18nDir, "core")
	if err = os.MkdirAll(i18nCoreDir, 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/core directory", err)
	}

	// Генерируем i18n/load.go
	var i18nContent string
	if i18nContent, err = renderTemplate("templates/i18n_load.go.tmpl", rootData); err != nil {
		return fmt.Errorf(i18n.Msg("failed to render i18n/load.go: %w"), err)
	}

	i18nLoadPath := filepath.Join(i18nDir, "load.go")
	if err = writeFile(i18nLoadPath, i18nContent); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/load.go", err)
	}

	langRuFile := filepath.Join(i18nCoreDir, "ru.json")
	var langRuData []byte
	if langRuData, err = templatesFS.ReadFile("templates/i18n_lang_ru.json"); err != nil {
		return fmt.Errorf(i18n.Msg("failed to read i18n_lang_ru.json from templates: %w"), err)
	}
	if err = os.WriteFile(langRuFile, langRuData, 0600); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/core/ru.json", err)
	}

	return
}

// CreateI18n: load.go и core/ru.json в директории i18n.
func (c *RootFilesCreator) CreateI18n(rootDir string, moduleName string) (err error) {

	if moduleName == "" {
		moduleName = DefaultModuleName
	}

	rootData := TemplateData{
		ModuleName: moduleName,
	}

	i18nDir := "i18n"
	if err = os.MkdirAll(i18nDir, 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n directory", err)
	}

	i18nCoreDir := filepath.Join(i18nDir, "core")
	if err = os.MkdirAll(i18nCoreDir, 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/core directory", err)
	}

	// Генерируем i18n/load.go
	var i18nContent string
	if i18nContent, err = renderTemplate("templates/i18n_load.go.tmpl", rootData); err != nil {
		return fmt.Errorf(i18n.Msg("failed to render i18n/load.go: %w"), err)
	}

	i18nLoadPath := filepath.Join(i18nDir, "load.go")
	if err = writeFile(i18nLoadPath, i18nContent); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/load.go", err)
	}

	langRuFile := filepath.Join(i18nCoreDir, "ru.json")
	var langRuData []byte
	if langRuData, err = templatesFS.ReadFile("templates/i18n_lang_ru.json"); err != nil {
		return fmt.Errorf(i18n.Msg("failed to read i18n_lang_ru.json from templates: %w"), err)
	}
	if err = os.WriteFile(langRuFile, langRuData, 0600); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/core/ru.json", err)
	}

	return
}
