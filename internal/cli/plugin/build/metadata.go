// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/loader"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func extractMetadata(ctx context.Context, outDir string, scopeName string, pluginDirs []string) (built []builtPlugin, err error) {

	if len(pluginDirs) == 0 {
		return
	}

	var wg sync.WaitGroup
	maxConcurrency := runtime.NumCPU()
	sem := make(chan struct{}, maxConcurrency)
	built = make([]builtPlugin, 0, len(pluginDirs))

	var mu sync.Mutex
	var firstErr error

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

			srcPath := filepath.Join(outDir, "plugin_"+d+".tgp")
			slog.Debug("loading plugin info", "plugin", d, "path", srcPath)
			info, loadErr := loader.LoadInfoFromTGP(ctx, scopeName, srcPath)
			if loadErr != nil {
				failure := newBuildFailure(d, phaseMetadata, loadErr, "")
				recordFirstFailure(ctx, &mu, &firstErr, failure)
				return
			}

			if info.Name == "" {
				failure := newBuildFailure(d, phaseMetadata, fmt.Errorf("%s", i18n.Msg("invalid plugin: missing name")), "")
				recordFirstFailure(ctx, &mu, &firstErr, failure)
				return
			}

			destPath := filepath.Join(outDir, info.Name+plugin.FileExtTGP)
			if renameErr := os.Rename(srcPath, destPath); renameErr != nil {
				failure := newBuildFailure(d, phaseMetadata, renameErr, "")
				recordFirstFailure(ctx, &mu, &firstErr, failure)
				return
			}

			checksumHex, sumErr := fileSHA256Hex(destPath)
			if sumErr != nil {
				failure := newBuildFailure(d, phaseMetadata, sumErr, "")
				recordFirstFailure(ctx, &mu, &firstErr, failure)
				return
			}

			mu.Lock()
			built = append(built, builtPlugin{
				Dir:      d,
				Name:     info.Name,
				Info:     info,
				TgpPath:  destPath,
				Checksum: checksumLine(checksumHex),
			})
			mu.Unlock()
		}(dir)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	return
}
