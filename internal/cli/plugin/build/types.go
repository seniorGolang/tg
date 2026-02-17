// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"io"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

type Params struct {
	Clean             bool
	OutDir            string
	RootDir           string
	Version           string
	ScopeName         string
	OutWriter         io.Writer // OutWriter != nil — вывод в виде markdown (таблица + блок кода), иначе — построчный текст.
	VersionLdVar      string
	OverrideManifest  string
	SkipVersionUpdate bool
}

type builtPlugin struct {
	Dir      string
	Name     string
	Info     plugin.Info
	TgpPath  string
	Checksum string
}
