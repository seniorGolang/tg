// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"io"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

type Params struct {
	RootDir           string
	OutDir            string
	Clean             bool
	OverrideManifest  string
	Version           string
	SkipVersionUpdate bool
	ScopeName         string
	VersionLdVar      string
	// OutWriter != nil — вывод в виде markdown (таблица + блок кода), иначе — построчный текст.
	OutWriter io.Writer
}

type builtPlugin struct {
	Dir      string
	Name     string
	TgpPath  string
	Checksum string
	Info     plugin.Info
}
