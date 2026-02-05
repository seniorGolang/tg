// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const gzipLevel = 9

func compressAll(ctx context.Context, outDir string, pluginDirs []string) (err error) {

	if len(pluginDirs) == 0 {
		return
	}

	maxConcurrency := runtime.NumCPU()
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

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

			path := filepath.Join(outDir, "plugin_"+d+".tgp")
			slog.Debug("compressing plugin", "plugin", d, "path", path)
			if runErr := gzipFileInPlace(path); runErr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = runErr
				}
				mu.Unlock()
				return
			}
		}(dir)
	}

	wg.Wait()
	return firstErr
}

func gzipFileInPlace(path string) (err error) {

	var f *os.File
	if f, err = os.Open(path); err != nil {
		return
	}
	defer f.Close()

	var data []byte
	if data, err = io.ReadAll(f); err != nil {
		return
	}
	f.Close()

	tmpPath := path + ".tmp"
	var out *os.File
	if out, err = os.Create(tmpPath); err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = out.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	var gz *gzip.Writer
	if gz, err = gzip.NewWriterLevel(out, gzipLevel); err != nil {
		return
	}

	if _, err = gz.Write(data); err != nil {
		_ = gz.Close()
		return
	}

	if err = gz.Close(); err != nil {
		return
	}

	if err = out.Close(); err != nil {
		return
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return
	}

	return
}
