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

func (u *URI) parse(spec string) (err error) {

	parts := strings.Split(spec, "@")
	specWithoutVersion := parts[0]
	if len(parts) > 1 {
		u.version.Original = parts[1]
		u.version.Original = strings.TrimSpace(u.version.Original)
		if u.version.Original != "" && !u.HasVersionConstraint() {
			var parsedVersion models.Version
			if parsedVersion, err = version.Parse(u.version.Original); err != nil {
				return
			}
			u.version = parsedVersion
		}
	}

	schemeSeparator := "://"
	schemeIndex := strings.Index(specWithoutVersion, schemeSeparator)
	if schemeIndex < 0 {
		return fmt.Errorf("URL must have a scheme")
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
		slashIndex := strings.Index(restAfterScheme, "/")
		hostPartEnd := len(restAfterScheme)
		if slashIndex >= 0 {
			hostPartEnd = slashIndex
		}

		if lastColonIndex < hostPartEnd {
			afterColon := restAfterScheme[lastColonIndex+1 : hostPartEnd]
			if isPortNumber(afterColon) {
				if slashIndex >= 0 {
					pathPart := restAfterScheme[slashIndex+1:]
					pathColonIndex := strings.LastIndex(pathPart, ":")
					if pathColonIndex >= 0 {
						extractedPackageName = pathPart[pathColonIndex+1:]
						urlPartAfterScheme = restAfterScheme[:slashIndex+1+pathColonIndex]
					}
				}
			} else {
				afterColonSuffix := restAfterScheme[lastColonIndex+1:]
				if len(afterColonSuffix) > 0 {
					urlPartAfterScheme = restAfterScheme[:lastColonIndex]
					extractedPackageName = afterColonSuffix
				}
			}
		} else {
			afterColonSuffix := restAfterScheme[lastColonIndex+1:]
			if len(afterColonSuffix) > 0 {
				urlPartAfterScheme = restAfterScheme[:lastColonIndex]
				extractedPackageName = afterColonSuffix
			}
		}
	}

	fullURL := scheme + schemeSeparator + urlPartAfterScheme
	parsedURL, parseErr := url.Parse(fullURL)
	if parseErr != nil {
		return fmt.Errorf("failed to parse URL: %w", parseErr)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must have a scheme")
	}

	u.parsedURL = parsedURL

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
