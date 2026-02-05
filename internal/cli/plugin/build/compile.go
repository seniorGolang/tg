// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func compileAll(ctx context.Context, rootDir string, outDir string, version string, pluginDirs []string, versionLdVar string) (err error) {

	if len(pluginDirs) == 0 {
		return
	}

	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	maxConcurrency := runtime.NumCPU()
	sem := make(chan struct{}, maxConcurrency)

	for _, dir := range pluginDirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			pluginDir := filepath.Join(rootDir, "plugins", d)
			outTgp := filepath.Join(outDir, "plugin_"+d+".tgp")

			slog.Debug("compiling plugin", "plugin", d, "dir", pluginDir, "out", outTgp)

			// version и versionLdVar приходят из конфига сборки, не от пользовательского ввода.
			// #nosec G204
			cmd := exec.CommandContext(ctx, "go", "build",
				"-ldflags", "-s -w -X "+versionLdVar+"="+version,
				"-o", outTgp,
				"-buildmode=c-shared",
				".")
			cmd.Dir = pluginDir
			cmd.Env = append(cmd.Environ(), "GOOS=wasip1", "GOARCH=wasm")

			if runErr := cmd.Run(); runErr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf(i18n.Msg("plugin %s failed: %w"), d, runErr)
				}
				mu.Unlock()
				return
			}
		}(dir)
	}

	wg.Wait()
	return firstErr
}
