// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// GoModManager управляет go.mod файлами.
type GoModManager struct{}

// Tidy запускает go mod tidy в корне проекта.
func (m *GoModManager) Tidy(ctx context.Context, rootDir string) (err error) {

	slog.Info(i18n.Msg("Running go mod tidy"))
	buildCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	buildCmd.Dir = rootDir
	var output []byte
	if output, err = buildCmd.CombinedOutput(); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to run go mod tidy: %w"), fmt.Errorf("%w\n%s", err, string(output)))
		return
	}

	return
}
