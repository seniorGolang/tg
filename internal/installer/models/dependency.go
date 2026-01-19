// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

import (
	"net/url"
	"strings"
)

// DependencyGraph представляет граф зависимостей.
type DependencyGraph struct {
	Nodes map[string]*DependencyNode
	Edges []*DependencyEdge
}

// DependencyNode представляет узел в графе зависимостей.
type DependencyNode struct {
	Package *Package
	Version Version
	ID      string
}

// DependencyEdge представляет ребро в графе зависимостей.
type DependencyEdge struct {
	From       *DependencyNode
	To         *DependencyNode
	Dependency *Dependency
}

// ParseDependencyString парсит строку зависимости в структуру Dependency.
// Поддерживаемые форматы:
// - package - зависимость без версии
// - package@version - зависимость с версией
// - source:package - зависимость из другого репозитория без версии
// - source:package@version - зависимость из другого репозитория с версией
// - URL:package@version - зависимость из URL репозитория
func ParseDependencyString(depStr string) (dep Dependency) {

	depStr = strings.TrimSpace(depStr)
	if depStr == "" {
		return
	}

	// Шаг 1: Разделяем по "@" для извлечения версии
	parts := strings.Split(depStr, "@")
	specWithoutVersion := parts[0]
	if len(parts) == 2 {
		dep.Version = parts[1]
	}

	// Шаг 2: Проверяем наличие ":" для разделения source и package
	colonIndex := strings.LastIndex(specWithoutVersion, ":")
	if colonIndex > 0 {
		// Проверяем, не является ли ":" частью схемы URL (://)
		schemeEndIndex := strings.Index(specWithoutVersion, "://")
		if schemeEndIndex < 0 || colonIndex > schemeEndIndex+2 {
			// Это не часть схемы, значит ":" разделяет source и package
			beforeColon := specWithoutVersion[:colonIndex]
			afterColon := specWithoutVersion[colonIndex+1:]

			// Проверяем, является ли часть до ":" валидным URL
			testURL := beforeColon
			if !strings.Contains(testURL, "://") {
				// Пробуем добавить схему для проверки
				testURL = "https://" + testURL
			}
			testParsedURL, testErr := url.Parse(testURL)
			if testErr == nil && testParsedURL.Scheme != "" {
				// Это валидный URL, значит это source
				// Восстанавливаем оригинальный URL без добавленной схемы
				if !strings.Contains(beforeColon, "://") {
					// Если не было схемы, используем https://
					dep.Source = "https://" + beforeColon
				} else {
					dep.Source = beforeColon
				}
				dep.Package = afterColon
				return
			}
		}
	}

	// Если не нашли source через ":", значит это просто package
	dep.Package = specWithoutVersion
	return
}
