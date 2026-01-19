// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

// parse разбирает строку спецификации на компоненты.
func (u *URI) parse(spec string) (err error) {

	// Шаг 1: Разделяем по "@" для извлечения версии
	parts := strings.Split(spec, "@")
	specWithoutVersion := parts[0]
	if len(parts) > 1 {
		u.version.Original = parts[1]
		u.version.Original = strings.TrimSpace(u.version.Original)
		if u.version.Original != "" && !u.hasVersionConstraint() {
			var parsedVersion models.Version
			if parsedVersion, err = version.Parse(u.version.Original); err != nil {
				// Игнорируем ошибку парсинга версии, оставляем version как строку
				err = nil
			} else {
				u.version = parsedVersion
			}
		}
	}

	// Шаг 2: Находим схему по паттерну "://" (схема должна быть всегда)
	schemeSeparator := "://"
	schemeIndex := strings.Index(specWithoutVersion, schemeSeparator)
	if schemeIndex < 0 {
		err = fmt.Errorf("URL must have a scheme")
		return
	}

	scheme := specWithoutVersion[:schemeIndex]
	restAfterScheme := specWithoutVersion[schemeIndex+len(schemeSeparator):]

	// Шаг 3: Извлекаем имя пакета из остальной части URL через ":".
	// Спецификация допускает два формата: scheme://host:port/path (порт — число) и scheme://source:packageName.
	// Последнее вхождение ":" может быть как портом (если после него число), так и разделителем source:packageName.
	// Проверка isPortNumber позволяет отличить эти случаи и при порте искать имя пакета в path.
	urlPartAfterScheme := restAfterScheme
	extractedPackageName := ""

	lastColonIndex := strings.LastIndex(restAfterScheme, ":")
	if lastColonIndex >= 0 {
		// Определяем, где заканчивается host:port часть (до первого "/")
		slashIndex := strings.Index(restAfterScheme, "/")
		hostPartEnd := len(restAfterScheme)
		if slashIndex >= 0 {
			hostPartEnd = slashIndex
		}

		// Если ":" находится в host части, проверяем, не порт ли это
		if lastColonIndex < hostPartEnd {
			afterColon := restAfterScheme[lastColonIndex+1 : hostPartEnd]
			if isPortNumber(afterColon) {
				// Это порт, ищем имя пакета в пути (после "/")
				if slashIndex >= 0 {
					pathPart := restAfterScheme[slashIndex+1:]
					pathColonIndex := strings.LastIndex(pathPart, ":")
					if pathColonIndex >= 0 {
						extractedPackageName = pathPart[pathColonIndex+1:]
						urlPartAfterScheme = restAfterScheme[:slashIndex+1+pathColonIndex]
					}
				}
			} else {
				// Это не порт, значит это имя пакета
				afterColon := restAfterScheme[lastColonIndex+1:]
				if len(afterColon) > 0 {
					urlPartAfterScheme = restAfterScheme[:lastColonIndex]
					extractedPackageName = afterColon
				}
			}
		} else {
			// ":" находится в пути, это имя пакета
			afterColon := restAfterScheme[lastColonIndex+1:]
			if len(afterColon) > 0 {
				urlPartAfterScheme = restAfterScheme[:lastColonIndex]
				extractedPackageName = afterColon
			}
		}
	}

	// Шаг 4: Собираем полный URL для парсинга (схема + остальная часть без имени пакета)
	fullURL := scheme + schemeSeparator + urlPartAfterScheme

	// Шаг 5: Парсим собранный URL
	parsedURL, parseErr := url.Parse(fullURL)
	if parseErr != nil {
		err = fmt.Errorf("failed to parse URL: %w", parseErr)
		return
	}

	if parsedURL.Scheme == "" {
		err = fmt.Errorf("URL must have a scheme")
		return
	}

	u.parsedURL = parsedURL

	// Сохраняем URL как source и извлеченное имя пакета (если есть)
	u.source = fullURL
	if extractedPackageName != "" {
		u.packageName = extractedPackageName
	}

	return
}

func isPortNumber(s string) (isPort bool) {

	_, err := strconv.Atoi(s)
	return err == nil
}
