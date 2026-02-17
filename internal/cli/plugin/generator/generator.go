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
	Kind                string // kind плагина: pre, stage, command, post (опционально)
	Author              string
	License             string
	Command             string
	Category            string
	DeployType          string
	PluginName          string // camelCase - для манифестов и приватных имен
	ModuleName          string
	Description         string
	PluginNameSnakeCase string // snake-case - для имен файлов
	PluginNameOriginal  string // оригинальное имя (для нормализации)
	PluginNameTitleCase string // TitleCase - только для экспортируемых типов Go
}

func getGitConfig(key string) (value string) {

	cmd := exec.Command("git", "config", "--get", key)
	var err error
	var output []byte
	if output, err = cmd.Output(); err != nil {
		return
	}
	return strings.TrimSpace(string(output))
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
	return name
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
		return errors.New(i18n.Msg("Name cannot be empty"))
	}
	if len(name) > MaxPluginNameLength {
		return fmt.Errorf(i18n.Msg("Name too long (maximum %d characters)"), MaxPluginNameLength)
	}

	normalizedName := normalizePluginName(name)
	for _, reserved := range ReservedPluginNames {
		if strings.EqualFold(normalizedName, reserved) {
			return fmt.Errorf(i18n.Msg("Name '%s' is reserved"), name)
		}
	}

	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return errors.New(i18n.Msg("Name can only contain letters, numbers, hyphens and underscores"))
		}
	}

	return
}

func toTitleCase(s string) (result string) {

	if len(s) == 0 {
		return s
	}
	parts := strings.Split(s, "-")
	partsResult := make([]string, len(parts))
	for i, part := range parts {
		if len(part) > 0 {
			partsResult[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(partsResult, "")
}

func toCamelCase(s string) (result string) {

	if len(s) == 0 {
		return s
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
	return strings.Join(partsResult, "")
}

func toSnakeCase(s string) (result string) {

	if len(s) == 0 {
		return s
	}
	return strings.ReplaceAll(strings.ToLower(s), "_", "-")
}

func normalizePluginName(name string) (normalized string) {

	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

func renderTemplate(templatePath string, data TemplateData) (content string, err error) {

	var contentBytes []byte
	if contentBytes, err = templatesFS.ReadFile(templatePath); err != nil {
		return
	}

	if strings.Contains(templatePath, "cicd_") {
		return string(contentBytes), nil
	}

	var tmpl *template.Template
	if tmpl, err = template.New("").Parse(string(contentBytes)); err != nil {
		return
	}

	var buf strings.Builder
	if err = tmpl.Execute(&buf, data); err != nil {
		return
	}

	return buf.String(), nil
}

func writeFile(path string, content string) (err error) {

	cleanPath := filepath.Clean(path)
	if err = os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return
	}

	finalContent := strings.TrimRight(content, "\n") + "\n"
	if err = os.WriteFile(cleanPath, []byte(finalContent), 0600); err != nil {
		return
	}

	return
}

// RunUpgrade обновляет все сгенерированные файлы (core и CI/CD).
func RunUpgrade(ctx context.Context, rootDir string, moduleName string, deployType string) (err error) {

	// Удаляем всю директорию core/
	coreDir := CoreDirName
	var coreStatErr error
	if _, coreStatErr = os.Stat(coreDir); coreStatErr == nil {
		if err = os.RemoveAll(coreDir); err != nil {
			return fmt.Errorf(i18n.Msg("failed to remove core directory: %w"), err)
		}
	}

	// Удаляем i18n/load.go
	i18nLoadPath := filepath.Join("i18n", "load.go")
	var i18nLoadStatErr error
	if _, i18nLoadStatErr = os.Stat(i18nLoadPath); i18nLoadStatErr == nil {
		if err = os.Remove(i18nLoadPath); err != nil {
			return fmt.Errorf(i18n.Msg("failed to remove i18n/load.go: %w"), err)
		}
	}

	// Удаляем i18n/core/ru.json
	i18nCoreRuPath := filepath.Join("i18n", "core", "ru.json")
	var i18nCoreRuStatErr error
	if _, i18nCoreRuStatErr = os.Stat(i18nCoreRuPath); i18nCoreRuStatErr == nil {
		if err = os.Remove(i18nCoreRuPath); err != nil {
			return fmt.Errorf(i18n.Msg("failed to remove i18n/core/ru.json: %w"), err)
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
							return fmt.Errorf(i18n.Msg("failed to remove CI/CD file: %w"), err)
						}
					}
				}
			}
		}
	}

	coreCreator := &CoreCreator{}
	if err = coreCreator.Create(rootDir, moduleName); err != nil {
		return fmt.Errorf(i18n.Msg("failed to recreate core: %w"), err)
	}

	rootFilesCreator := &RootFilesCreator{}
	if err = rootFilesCreator.CreateI18n(rootDir, moduleName); err != nil {
		return fmt.Errorf(i18n.Msg("failed to recreate i18n files: %w"), err)
	}

	if detectedDeployType != DeployTypeNone {
		if err = cicdCreator.Create(rootDir, detectedDeployType); err != nil {
			return fmt.Errorf(i18n.Msg("failed to recreate CI/CD: %w"), err)
		}
	}

	var loader pluginLoader
	var loaderErr error
	if loader, loaderErr = createLoader(""); loaderErr == nil {
		var listErr error
		var pluginNames []string
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
		return fmt.Errorf(i18n.Msg("failed to run go mod tidy: %w"), err)
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
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "core", err)
		}
	}

	// Нормализуем имя плагина
	normalizedName := normalizePluginName(name)
	pluginDir := filepath.Clean(filepath.Join(PluginsDirName, normalizedName))
	if pluginCreator.Exists(pluginDir) {
		return fmt.Errorf(i18n.Msg("plugin %s already exists"), normalizedName)
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
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "plugin", err)
	}

	if !coreExists {
		if err = rootFilesCreator.Create(rootDir, data, moduleName); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "root files", err)
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
				return fmt.Errorf(i18n.Msg("failed to render gitignore template: %w"), err)
			}
			if err = writeFile(gitignorePath, content); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), ".gitignore", err)
			}
		}
	}

	if deployType != DeployTypeNone && !coreExists {
		if err = cicdCreator.Create(rootDir, deployType); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "CI/CD configuration", err)
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
		return fmt.Errorf(i18n.Msg("failed to run go mod tidy: %w"), err)
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
		return errors.New(i18n.Msg("core module not found, run plugin init first"))
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
		return fmt.Errorf(i18n.Msg("directory %s already exists"), pluginDir)
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
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "plugin", err)
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
		return errors.New(i18n.Msg("go.mod not found in project root"))
	}

	distDir := filepath.Clean(filepath.Join(pluginPath, DistDirName))
	if err = os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "dist directory", err)
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
		return fmt.Errorf(i18n.Msg("compilation error: %w\n%s"), err, string(output))
	}

	// Генерируем checksum
	var pluginBytes []byte
	if pluginBytes, err = os.ReadFile(outputFile); err != nil {
		return fmt.Errorf(i18n.Msg("failed to read plugin file: %w"), err)
	}

	hash := sha256.Sum256(pluginBytes)
	checksumFile := filepath.Clean(filepath.Join(distDir, fmt.Sprintf("%s%s", pluginName, ChecksumExtension)))
	if err = os.WriteFile(checksumFile, []byte(hex.EncodeToString(hash[:])), 0600); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "checksum", err)
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
