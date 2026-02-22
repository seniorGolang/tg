// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package fs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/wasm/env"
)

type envResolver struct {
	cache map[string]string
}

func newEnvResolver() (resolver *envResolver) {

	return &envResolver{
		cache: make(map[string]string),
	}
}

// resolve разрешает переменную окружения с кэшированием в пределах вызова.
// Использует env.GetValue (go env -json для Go-переменных, os.Getenv для остальных).
func (r *envResolver) resolve(key string) (value string, ok bool) {

	if cached, found := r.cache[key]; found {
		return cached, true
	}

	value, ok = env.GetValue(key)
	if !ok {
		return "", false
	}

	r.cache[key] = value
	return
}

type tgPathResolver struct {
	basePath string
}

// basePath берётся из customPath (wasm.WithTGPath / fs.NewBuilder); если не передан — пустая строка, @tg/ не резолвится.
func newTGPathResolver(customPath string) (resolver *tgPathResolver) {

	return &tgPathResolver{
		basePath: customPath,
	}
}

// resolve разрешает маркер @tg в путь относительно папки настроек.
func (r *tgPathResolver) resolve(path string) (resolvedPath string) {

	trimmed := strings.TrimPrefix(path, pathPrefixTG)
	if trimmed == "" {
		return r.basePath
	}

	relativePath := strings.TrimPrefix(trimmed, "/")
	return filepath.Join(r.basePath, relativePath)
}

type rootPathResolver struct {
	rootDir string
}

func newRootPathResolver(rootDir string) (resolver *rootPathResolver) {

	return &rootPathResolver{
		rootDir: rootDir,
	}
}

// resolve разрешает маркер @root в путь относительно rootDir.
func (r *rootPathResolver) resolve(path string) (resolvedPath string) {

	trimmed := strings.TrimPrefix(path, pathPrefixRoot)
	if trimmed == "" {
		return r.rootDir
	}

	relativePath := strings.TrimPrefix(trimmed, "/")
	return filepath.Join(r.rootDir, relativePath)
}

// findGoModRoot находит корневую директорию Go модуля, ища go.mod начиная с startDir и поднимаясь вверх.
func findGoModRoot(startDir string) (rootDir string, found bool) {

	var err error
	var currentDir string
	if currentDir, err = filepath.Abs(startDir); err != nil {
		return
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		var fileInfo os.FileInfo
		if fileInfo, err = os.Stat(goModPath); err == nil && !fileInfo.IsDir() {
			rootDir = currentDir
			found = true
			return
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return
		}

		currentDir = parentDir
	}
}

type goPathResolver struct {
	rootDir string
}

func newGoPathResolver(rootDir string) (resolver *goPathResolver) {

	return &goPathResolver{
		rootDir: rootDir,
	}
}

// resolve разрешает маркер @go в путь относительно корня Go модуля.
func (r *goPathResolver) resolve(path string) (resolvedPath string) {

	var found bool
	var goModRoot string
	goModRoot, found = findGoModRoot(r.rootDir)
	if !found {
		return ""
	}

	trimmed := strings.TrimPrefix(path, pathPrefixGo)
	if trimmed == "" {
		return goModRoot
	}

	relativePath := strings.TrimPrefix(trimmed, "/")
	return filepath.Join(goModRoot, relativePath)
}

// expandHome расширяет ~ в домашнюю директорию.
func expandHome(path string) (expandedPath string, err error) {

	if !strings.HasPrefix(path, "~") {
		expandedPath = path
		return
	}

	var homeDir string
	if homeDir, err = os.UserHomeDir(); err != nil {
		return
	}

	if path == "~" {
		expandedPath = homeDir
		return
	}

	if len(path) > 1 && (path[1] == '/' || path[1] == filepath.Separator) {
		expandedPath = filepath.Join(homeDir, path[2:])
		return
	}

	return filepath.Join(homeDir, path[1:]), nil
}
