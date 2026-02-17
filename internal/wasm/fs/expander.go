// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import (
	"log/slog"
	"path/filepath"
	"strings"
)

// pathExpander расширяет пути с учетом их типа.
type pathExpander struct {
	envResolver  *envResolver
	tgResolver   *tgPathResolver
	rootResolver *rootPathResolver
	goResolver   *goPathResolver
}

func newPathExpander(rootDir string, tgPath string) (expander *pathExpander) {
	return &pathExpander{
		envResolver:  newEnvResolver(),
		tgResolver:   newTGPathResolver(tgPath),
		rootResolver: newRootPathResolver(rootDir),
		goResolver:   newGoPathResolver(rootDir),
	}
}

// expand расширяет путь в зависимости от его типа.
func (e *pathExpander) expand(path string) (expandedPath string, err error) {

	pathType := detectPathType(path)

	switch pathType {
	case PathTypeGo:
		if expandedPath = e.goResolver.resolve(path); expandedPath == "" {
			slog.Warn("go path expanded path is empty")
		}
		return

	case PathTypeRoot:
		return e.rootResolver.resolve(path), nil

	case PathTypeTG:
		return e.tgResolver.resolve(path), nil

	case PathTypeEnv:
		return e.expandEnv(path)

	case PathTypeHome:
		return expandHome(path)

	case PathTypeAbsolute:
		expandedPath = filepath.Clean(path)
		return

	default:
		expandedPath = filepath.Clean(path)
		return
	}
}

// expandEnv расширяет переменные окружения в пути.
func (e *pathExpander) expandEnv(path string) (expandedPath string, err error) {

	// Извлекаем переменные из пути и заменяем их
	result := osExpandEnv(path, e.envResolver)
	if result == "" {
		return "", nil
	}

	// Нормализуем путь
	expandedPath = filepath.Clean(result)

	// Делаем путь абсолютным, если он еще не абсолютный
	if !filepath.IsAbs(expandedPath) {
		var absErr error
		expandedPath, absErr = filepath.Abs(expandedPath)
		if absErr != nil {
			return "", absErr
		}
	}

	return
}

// osExpandEnv расширяет переменные окружения в строке с использованием резолвера.
func osExpandEnv(s string, resolver *envResolver) (result string) {

	var resultBuilder strings.Builder
	var i int

	for i < len(s) {
		if s[i] == '$' {
			varName, consumed := extractVarName(s[i:])
			if varName != "" {
				if value, ok := resolver.resolve(varName); ok {
					resultBuilder.WriteString(value)
				}
				i += consumed
				continue
			}
		}

		resultBuilder.WriteByte(s[i])
		i++
	}

	result = resultBuilder.String()
	return
}

// extractVarName извлекает имя переменной из строки, начиная с $.
func extractVarName(s string) (varName string, consumed int) {

	if len(s) < 2 {
		return "", 0
	}

	if s[1] == '{' {
		// Формат ${VAR}
		end := strings.IndexByte(s[2:], '}')
		if end == -1 {
			return "", 0
		}
		varName = s[2 : end+2]
		consumed = end + 3
		return
	}

	// Формат $VAR
	var i int
	for i = 1; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			break
		}
	}

	if i == 1 {
		return "", 0
	}

	varName = s[1:i]
	consumed = i
	return
}
