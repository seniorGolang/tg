// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	phaseCompile  = "compile"
	phaseCompress = "compress"
	phaseMetadata = "metadata"
)

type BuildFailure struct {
	Plugin string
	Phase  string
	Cause  error
	Output string
}

func (f BuildFailure) Error() (message string) {

	message = fmt.Sprintf(i18n.Msg("plugin %s failed during %s: %v"), f.Plugin, f.Phase, f.Cause)
	return
}

func (f BuildFailure) Unwrap() (cause error) {

	return f.Cause
}

func newBuildFailure(plugin string, phase string, cause error, output string) (failure *BuildFailure) {

	failure = &BuildFailure{
		Plugin: plugin,
		Phase:  phase,
		Cause:  cause,
		Output: output,
	}
	return
}

func logBuildFailure(ctx context.Context, failure *BuildFailure) {

	attrs := []any{
		slog.String("plugin", failure.Plugin),
		slog.String("phase", failure.Phase),
		slog.String("error", failure.Cause.Error()),
	}
	if failure.Output != "" {
		attrs = append(attrs, slog.String("output", failure.Output))
	}
	slog.ErrorContext(ctx, i18n.Msg("Plugin build failed"), attrs...)
}

func recordFirstFailure(ctx context.Context, mu *sync.Mutex, firstErr *error, failure *BuildFailure) {

	mu.Lock()
	defer mu.Unlock()
	if *firstErr != nil {
		return
	}
	logBuildFailure(ctx, failure)
	*firstErr = failure
}
