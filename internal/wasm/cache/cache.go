// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cache

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/installer/storage"

	"github.com/tetratelabs/wazero"
)

var (
	compilationCache wazero.CompilationCache
	cacheOnce        sync.Once
	cacheErr         error
)

func GetCompilationCache(ctx context.Context) (cache wazero.CompilationCache, err error) {

	cacheOnce.Do(func() {
		var scopeName string
		var scopeErr error
		scopeName, scopeErr = storage.GetEffectiveScope()
		if scopeErr != nil {
			cacheErr = fmt.Errorf("failed to get current scope: %w", scopeErr)
			return
		}

		cacheDir := storage.GetCacheDir(scopeName)

		var mkdirErr error
		if mkdirErr = os.MkdirAll(cacheDir, storage.FilePermDir); mkdirErr != nil {
			cacheErr = fmt.Errorf("failed to create cache directory: %w", mkdirErr)
			return
		}

		compilationCache, cacheErr = wazero.NewCompilationCacheWithDir(cacheDir)
		if cacheErr != nil {
			cacheErr = fmt.Errorf("failed to create compilation cache: %w", cacheErr)
			return
		}
	})

	cache = compilationCache
	err = cacheErr
	return
}

func CloseCompilationCache(ctx context.Context) (err error) {

	if compilationCache != nil {
		return compilationCache.Close(ctx)
	}
	return
}
