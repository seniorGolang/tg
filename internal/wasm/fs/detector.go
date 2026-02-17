// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import "strings"

const (
	// pathPrefixGo - префикс пути для корня Go модуля.
	pathPrefixGo = "@go"
	// pathPrefixRoot - префикс пути для корня проекта.
	pathPrefixRoot = "@root"
	// pathPrefixTG - префикс пути для папки настроек.
	pathPrefixTG = "@tg"
)

// PathType представляет тип пути.
type PathType int

const (
	// PathTypeGo - путь относительно корня Go модуля (@go).
	PathTypeGo PathType = iota
	// PathTypeRoot - относительный путь относительно rootDir (@root).
	PathTypeRoot
	// PathTypeTG - путь относительно папки настроек (@tg).
	PathTypeTG
	// PathTypeEnv - переменная окружения ($VAR или ${VAR}).
	PathTypeEnv
	// PathTypeHome - домашняя директория (~).
	PathTypeHome
	// PathTypeAbsolute - абсолютный путь.
	PathTypeAbsolute
)

// detectPathType определяет тип пути по префиксу.
func detectPathType(path string) (pathType PathType) {

	if strings.HasPrefix(path, pathPrefixGo) {
		pathType = PathTypeGo
		return
	}

	if strings.HasPrefix(path, pathPrefixRoot) {
		pathType = PathTypeRoot
		return
	}

	if strings.HasPrefix(path, pathPrefixTG) {
		pathType = PathTypeTG
		return
	}

	if strings.HasPrefix(path, "$") || strings.HasPrefix(path, "${") {
		pathType = PathTypeEnv
		return
	}

	if strings.HasPrefix(path, "~") {
		pathType = PathTypeHome
		return
	}

	pathType = PathTypeAbsolute
	return
}
