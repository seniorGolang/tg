// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

//go:embed templates/*
var templatesFS embed.FS

// TemplateData содержит данные для шаблонов.
type TemplateData struct {
	PluginName          string // camelCase - для манифестов и приватных имен
	PluginNameTitleCase string // TitleCase - только для экспортируемых типов Go
	PluginNameSnakeCase string // snake-case - для имен файлов
	PluginNameOriginal  string // оригинальное имя (для нормализации)
	Description         string
	Author              string
	License             string
	Category            string
	Command             string
	ModuleName          string
	DeployType          string
	Kind                string // kind плагина: pre, stage, command, post (опционально)
}

func getGitConfig(key string) (value string) {

	cmd := exec.Command("git", "config", "--get", key)
	var output []byte
	var err error
	if output, err = cmd.Output(); err != nil {
		return
	}
	value = strings.TrimSpace(string(output))
	return
}

func getGitAuthor() (author string) {

	name := getGitConfig("user.name")
	email := getGitConfig("user.email")
	if name == "" && email == "" {
		author = DefaultAuthor
		return
	}
	if email != "" {
		author = name + " <" + email + ">"
		return
	}
	author = name
	return
}

const (
	// MaxPluginNameLength максимальная длина имени плагина.
	MaxPluginNameLength = 50
)

var (
	// ReservedPluginNames зарезервированные имена, которые нельзя использовать для плагинов.
	ReservedPluginNames = []string{CoreDirName, PluginsDirName, DistDirName, GoModFileName, GoSumFileName}
)

func isValidPluginName(name string) (err error) {

	if len(name) == 0 {
		err = errors.New(i18n.Msg("Name cannot be empty"))
		return
	}
	if len(name) > MaxPluginNameLength {
		err = fmt.Errorf(i18n.Msg("Name too long (maximum %d characters)"), MaxPluginNameLength)
		return
	}

	normalizedName := normalizePluginName(name)
	for _, reserved := range ReservedPluginNames {
		if strings.EqualFold(normalizedName, reserved) {
			err = fmt.Errorf(i18n.Msg("Name '%s' is reserved"), name)
			return
		}
	}

	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			err = errors.New(i18n.Msg("Name can only contain letters, numbers, hyphens and underscores"))
			return
		}
	}

	return
}

// toTitleCase преобразует строку в TitleCase.
func toTitleCase(s string) (result string) {

	if len(s) == 0 {
		result = s
		return
	}
	parts := strings.Split(s, "-")
	partsResult := make([]string, len(parts))
	for i, part := range parts {
		if len(part) > 0 {
			partsResult[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	result = strings.Join(partsResult, "")
	return
}

// toCamelCase преобразует строку в camelCase.
func toCamelCase(s string) (result string) {

	if len(s) == 0 {
		result = s
		return
	}
	parts := strings.Split(s, "-")
	partsResult := make([]string, len(parts))
	for i, part := range parts {
		if i == 0 {
			partsResult[i] = strings.ToLower(part)
		} else if len(part) > 0 {
			partsResult[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	result = strings.Join(partsResult, "")
	return
}

// toSnakeCase преобразует строку в snake-case.
func toSnakeCase(s string) (result string) {

	if len(s) == 0 {
		result = s
		return
	}
	// Нормализуем: приводим к lowercase и заменяем подчеркивания на дефисы
	result = strings.ToLower(s)
	result = strings.ReplaceAll(result, "_", "-")
	return
}

// normalizePluginName нормализует имя плагина (приводит к lowercase, заменяет подчеркивания на дефисы).
func normalizePluginName(name string) (normalized string) {

	normalized = strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return
}

// renderTemplate рендерит шаблон из embed FS.
func renderTemplate(templatePath string, data TemplateData) (content string, err error) {

	var contentBytes []byte
	if contentBytes, err = templatesFS.ReadFile(templatePath); err != nil {
		return
	}

	// Для CI/CD шаблонов используем простую замену без парсинга как Go template
	if strings.Contains(templatePath, "cicd_") {
		content = string(contentBytes)
		return
	}

	var tmpl *template.Template
	if tmpl, err = template.New("").Parse(string(contentBytes)); err != nil {
		return
	}

	var buf strings.Builder
	if err = tmpl.Execute(&buf, data); err != nil {
		return
	}

	content = buf.String()
	return
}

func writeFile(path string, content string) (err error) {

	// Нормализуем путь для безопасности
	cleanPath := filepath.Clean(path)
	if err = os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return
	}

	// Нормализуем конец файла: убираем лишние пустые строки, но оставляем одну
	finalContent := strings.TrimRight(content, "\n") + "\n"

	err = os.WriteFile(cleanPath, []byte(finalContent), 0600)
	return
}

// RunUpgrade обновляет все сгенерированные файлы (core и CI/CD).
func RunUpgrade(ctx context.Context, rootDir string, moduleName string, deployType string) (err error) {

	// Удаляем всю директорию core/
	coreDir := CoreDirName
	var coreStatErr error
	if _, coreStatErr = os.Stat(coreDir); coreStatErr == nil {
		if err = os.RemoveAll(coreDir); err != nil {
			err = fmt.Errorf(i18n.Msg("failed to remove core directory: %w"), err)
			return
		}
	}

	// Удаляем i18n/load.go
	i18nLoadPath := filepath.Join("i18n", "load.go")
	var i18nLoadStatErr error
	if _, i18nLoadStatErr = os.Stat(i18nLoadPath); i18nLoadStatErr == nil {
		if err = os.Remove(i18nLoadPath); err != nil {
			err = fmt.Errorf(i18n.Msg("failed to remove i18n/load.go: %w"), err)
			return
		}
	}

	// Удаляем i18n/core/ru.json
	i18nCoreRuPath := filepath.Join("i18n", "core", "ru.json")
	var i18nCoreRuStatErr error
	if _, i18nCoreRuStatErr = os.Stat(i18nCoreRuPath); i18nCoreRuStatErr == nil {
		if err = os.Remove(i18nCoreRuPath); err != nil {
			err = fmt.Errorf(i18n.Msg("failed to remove i18n/core/ru.json: %w"), err)
			return
		}
	}

	// Удаляем CI/CD файлы, если они сгенерированы
	cicdCreator := &CICDCreator{}
	detectedDeployType := cicdCreator.DetectDeployType(rootDir)
	if detectedDeployType != DeployTypeNone {
		var cicdPath string
		switch detectedDeployType {
		case DeployTypeGitLab:
			cicdPath = GitLabCIFileName
		case DeployTypeGitHub:
			cicdPath = filepath.Join(GitHubWorkflowsDir, GitHubWorkflowsSubDir, GitHubDeployFileName)
		}
		if cicdPath != "" {
			var cicdStatErr error
			if _, cicdStatErr = os.Stat(cicdPath); cicdStatErr == nil {
				// Проверяем, что файл сгенерирован
				var cicdData []byte
				var readErr error
				if cicdData, readErr = os.ReadFile(cicdPath); readErr == nil {
					cicdContent := string(cicdData)
					firstLine := strings.Split(cicdContent, "\n")[0]
					if strings.HasPrefix(firstLine, GeneratedCommentYAML) {
						if err = os.Remove(cicdPath); err != nil {
							err = fmt.Errorf(i18n.Msg("failed to remove CI/CD file: %w"), err)
							return
						}
					}
				}
			}
		}
	}

	coreCreator := &CoreCreator{}
	if err = coreCreator.Create(rootDir, moduleName); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to recreate core: %w"), err)
		return
	}

	rootFilesCreator := &RootFilesCreator{}
	if err = rootFilesCreator.CreateI18n(rootDir, moduleName); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to recreate i18n files: %w"), err)
		return
	}

	if detectedDeployType != DeployTypeNone {
		if err = cicdCreator.Create(rootDir, detectedDeployType); err != nil {
			err = fmt.Errorf(i18n.Msg("failed to recreate CI/CD: %w"), err)
			return
		}
	}

	var loader pluginLoader
	var loaderErr error
	if loader, loaderErr = createLoader(""); loaderErr == nil {
		var pluginNames []string
		var listErr error
		if pluginNames, listErr = GetInitGenerators(loader); listErr == nil && len(pluginNames) > 0 {
			validPluginNames := ResolveConflictsForNames(loader, pluginNames)
			if len(validPluginNames) > 0 {
				if cleanupErr := ExecuteInitCleanup(ctx, loader, validPluginNames, rootDir); cleanupErr != nil {
					slog.Warn(i18n.Msg("Failed to execute init cleanup"), "error", cleanupErr)
				}
				if execErr := ExecuteInitGenerators(ctx, loader, validPluginNames, rootDir, moduleName); execErr != nil {
					slog.Warn(i18n.Msg("Failed to execute init generators"), "error", execErr)
				}
			}
		}
	}

	goModManager := &GoModManager{}
	if err = goModManager.Tidy(ctx, rootDir); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to run go mod tidy: %w"), err)
		return
	}

	return
}

// RunInit выполняет команду init.
func RunInit(ctx context.Context, rootDir string, name, command, deployType, license, moduleName, kind string) (err error) {

	validator := &Validator{}
	params := &PluginParams{
		Name:       name,
		Command:    command,
		DeployType: deployType,
		License:    license,
		ModuleName: moduleName,
		Kind:       kind,
	}
	if err = validator.ValidateAndNormalize(params); err != nil {
		return
	}

	name = params.Name
	command = params.Command
	deployType = params.DeployType
	license = params.License
	moduleName = params.ModuleName
	kind = params.Kind

	author := getGitAuthor()
	if author == "" {
		author = DefaultAuthor
	}

	coreCreator := &CoreCreator{}
	pluginCreator := &PluginCreator{}
	rootFilesCreator := &RootFilesCreator{}
	cicdCreator := &CICDCreator{}
	goModManager := &GoModManager{}

	coreExists := coreCreator.Exists()

	if !coreExists {
		if err = coreCreator.Create(rootDir, moduleName); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "core", err)
			return
		}
	}

	// Нормализуем имя плагина
	normalizedName := normalizePluginName(name)
	pluginDir := filepath.Clean(filepath.Join(PluginsDirName, normalizedName))
	if pluginCreator.Exists(pluginDir) {
		err = fmt.Errorf(i18n.Msg("plugin %s already exists"), normalizedName)
		return
	}

	data := TemplateData{
		PluginName:          toCamelCase(normalizedName), // camelCase для манифестов
		PluginNameTitleCase: toTitleCase(normalizedName), // TitleCase для экспортируемых типов
		PluginNameSnakeCase: toSnakeCase(normalizedName), // snake-case для файлов
		PluginNameOriginal:  normalizedName,              // нормализованное имя
		Description:         "This is an plugin",
		Author:              author,
		License:             license,
		Category:            DefaultCategory,
		Command:             command,
		ModuleName:          moduleName,
		DeployType:          deployType,
		Kind:                kind,
	}

	if err = pluginCreator.Create(pluginDir, data); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "plugin", err)
		return
	}

	if !coreExists {
		if err = rootFilesCreator.Create(rootDir, data, moduleName); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "root files", err)
			return
		}
	} else {
		// Если core уже существует, создаём только .gitignore если его нет
		gitignorePath := ".gitignore"
		var statErr error
		if _, statErr = os.Stat(gitignorePath); os.IsNotExist(statErr) {
			rootData := TemplateData{
				ModuleName: moduleName,
			}
			var content string
			if content, err = renderTemplate("templates/gitignore.tmpl", rootData); err != nil {
				err = fmt.Errorf(i18n.Msg("failed to render gitignore template: %w"), err)
				return
			}
			if err = writeFile(gitignorePath, content); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), ".gitignore", err)
				return
			}
		}
	}

	if deployType != DeployTypeNone && !coreExists {
		if err = cicdCreator.Create(rootDir, deployType); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "CI/CD configuration", err)
			return
		}
	}

	// Генерация от плагинов с InitPkgs — после создания go.mod, до go mod tidy (плагинам нужен go.mod для импортов).
	goModPath := filepath.Join(rootDir, GoModFileName)
	var statErr error
	if _, statErr = os.Stat(goModPath); os.IsNotExist(statErr) {
		slog.Debug(i18n.Msg("go.mod not found, skipping plugin generation"),
			"goModPath", goModPath)
	} else {
		var loader pluginLoader
		var loaderErr error
		if loader, loaderErr = createLoader(""); loaderErr == nil {
			var pluginNames []string
			var listErr error
			if pluginNames, listErr = GetInitGenerators(loader); listErr == nil && len(pluginNames) > 0 {
				validPluginNames := ResolveConflictsForNames(loader, pluginNames)
				if len(validPluginNames) > 0 {
					slog.Debug(i18n.Msg("Executing init generators after file creation"),
						"pluginsCount", len(validPluginNames),
						"goModPath", goModPath)
					if execErr := ExecuteInitGenerators(ctx, loader, validPluginNames, rootDir, moduleName); execErr != nil {
						slog.Warn(i18n.Msg("Failed to execute init generators"), "error", execErr)
					}
				}
			}
		}
	}

	if err = goModManager.Tidy(ctx, rootDir); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to run go mod tidy: %w"), err)
		return
	}

	// Сообщение о успехе выводится через logger в вызывающем коде
	return
}

// RunAdd выполняет команду add.
func RunAdd(rootDir string, name, command, dir, license, moduleName, kind string) (err error) {

	validator := &Validator{}
	params := &PluginParams{
		Name:       name,
		Command:    command,
		DeployType: "none", // Для add команды deployType не используется
		License:    license,
		ModuleName: moduleName,
		Kind:       kind,
	}
	if err = validator.ValidateAndNormalize(params); err != nil {
		return
	}

	// Используем нормализованные значения
	name = params.Name
	command = params.Command
	license = params.License
	moduleName = params.ModuleName
	kind = params.Kind

	// Проверяем существование core/
	coreCreator := &CoreCreator{}
	if !coreCreator.Exists() {
		err = errors.New(i18n.Msg("core module not found, run plugin init first"))
		return
	}

	author := getGitAuthor()
	if author == "" {
		author = DefaultAuthor
	}

	// Нормализуем имя плагина
	normalizedName := normalizePluginName(name)

	// Если dir не указан, используем значение по умолчанию как в init
	pluginDir := dir
	if pluginDir == "" {
		pluginDir = filepath.Clean(filepath.Join(PluginsDirName, normalizedName))
	} else {
		// Нормализуем указанный путь
		pluginDir = filepath.Clean(pluginDir)
	}

	pluginCreator := &PluginCreator{}
	if pluginCreator.Exists(pluginDir) {
		err = fmt.Errorf(i18n.Msg("directory %s already exists"), pluginDir)
		return
	}

	data := TemplateData{
		PluginName:          toCamelCase(normalizedName), // camelCase для манифестов
		PluginNameTitleCase: toTitleCase(normalizedName), // TitleCase для экспортируемых типов
		PluginNameSnakeCase: toSnakeCase(normalizedName), // snake-case для файлов
		PluginNameOriginal:  normalizedName,              // нормализованное имя
		Description:         fmt.Sprintf(i18n.Msg("Plugin %s"), normalizedName),
		Author:              author,
		License:             license,
		Category:            DefaultCategory,
		Command:             command,
		ModuleName:          moduleName,
		DeployType:          DeployTypeNone, // Для add команды deployType не используется
		Kind:                kind,
	}

	if err = pluginCreator.Create(pluginDir, data); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "plugin", err)
		return
	}

	// Сообщение о успехе выводится через logger в вызывающем коде
	return
}

// RunBuild выполняет команду build.
func RunBuild(ctx context.Context, rootDir string, pluginDir string) (err error) {

	// В WASM файловая система монтируется в корень "/", поэтому используем относительные пути
	// Определяем путь к плагину относительно корня проекта
	var pluginPath string
	if pluginDir == "" {
		// Если не указан, пытаемся определить из текущей директории
		// В WASM os.Getwd() вернет "/", поэтому нужно использовать относительные пути
		pluginPath = "."
	} else {
		pluginPath = pluginDir
	}

	// Нормализуем путь к плагину
	pluginPath = filepath.Clean(pluginPath)

	// Определяем имя плагина из пути и нормализуем
	pluginNameOriginal := filepath.Base(pluginPath)
	normalizedName := normalizePluginName(pluginNameOriginal)
	pluginName := toCamelCase(normalizedName)

	// Проверяем наличие go.mod в корне проекта
	rootGoMod := GoModFileName
	var statErr error
	if _, statErr = os.Stat(rootGoMod); os.IsNotExist(statErr) {
		err = errors.New(i18n.Msg("go.mod not found in project root"))
		return
	}

	distDir := filepath.Clean(filepath.Join(pluginPath, DistDirName))
	if err = os.MkdirAll(distDir, 0755); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "dist directory", err)
		return
	}

	// Компилируем плагин из корня проекта с указанием пути к плагину
	// Используем camelCase для имени файла
	outputFile := filepath.Clean(filepath.Join(distDir, fmt.Sprintf("%s%s", pluginName, PluginExtension)))
	// Сообщение о компиляции выводится через logger в вызывающем коде

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outputFile, "-buildmode=c-shared", "./"+pluginPath)
	buildCmd.Dir = "." // В WASM корень проекта - это текущая директория
	buildCmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")

	var output []byte
	if output, err = buildCmd.CombinedOutput(); err != nil {
		err = fmt.Errorf(i18n.Msg("compilation error: %w\n%s"), err, string(output))
		return
	}

	// Генерируем checksum
	var pluginBytes []byte
	if pluginBytes, err = os.ReadFile(outputFile); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to read plugin file: %w"), err)
		return
	}

	hash := sha256.Sum256(pluginBytes)
	checksumFile := filepath.Clean(filepath.Join(distDir, fmt.Sprintf("%s%s", pluginName, ChecksumExtension)))
	if err = os.WriteFile(checksumFile, []byte(hex.EncodeToString(hash[:])), 0600); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "checksum", err)
		return
	}

	// Копируем plugin.json из директории плагина с именем плагина
	pluginJson := filepath.Join(pluginPath, PluginJSONFileName)
	distJson := filepath.Join(distDir, fmt.Sprintf("%s%s", pluginName, JSONExtension))
	var jsonStatErr error
	if _, jsonStatErr = os.Stat(pluginJson); jsonStatErr == nil {
		var jsonBytes []byte
		var readErr error
		if jsonBytes, readErr = os.ReadFile(pluginJson); readErr == nil {
			// Пытаемся записать plugin.json, но не критично для генерации
			// Если не удалось скопировать, плагин всё равно будет работать
			_ = os.WriteFile(distJson, jsonBytes, 0600)
		}
	}

	// Сообщение о успехе выводится через logger в вызывающем коде
	return
}
