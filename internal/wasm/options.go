// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package wasm

import (
	"github.com/seniorGolang/tg/v3/internal/wasm/host"

	"github.com/tetratelabs/wazero"
)

// hostOptions представляет опции для создания Host.
type hostOptions struct {
	// CompilationCache - опциональный кэш компиляции для переиспользования скомпилированных модулей.
	// Если nil, кэш не используется.
	CompilationCache wazero.CompilationCache

	// TLSConfig - конфигурация TLS для сетевых соединений.
	// Если nil, используется конфигурация по умолчанию.
	TLSConfig *host.TLSConfig

	// TGPath - путь к папке настроек для маркера @tg/.
	// Если пусто, используется путь по умолчанию (~/.config/tg или $XDG_CONFIG_HOME/tg).
	TGPath string

	// MuteLogs отключает вывод сообщений уровня debug в wasm.
	MuteLogs bool
}

// Option представляет функцию опции для настройки hostOptions.
type Option func(opts *hostOptions)

func WithTLSConfig(tlsConfig host.TLSConfig) (opt Option) {
	return func(opts *hostOptions) {
		opts.TLSConfig = &tlsConfig
	}
}

func WithCompilationCache(cache wazero.CompilationCache) (opt Option) {
	return func(opts *hostOptions) {
		opts.CompilationCache = cache
	}
}

func WithTGPath(tgPath string) (opt Option) {
	return func(opts *hostOptions) {
		opts.TGPath = tgPath
	}
}

func MuteLogs() (opt Option) {
	return func(opts *hostOptions) {
		opts.MuteLogs = true
	}
}
