// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	// maxFileSize ограничивает размер файла при распаковке для защиты от decompression bomb
	maxFileSize     = 100 * 1024 * 1024
	archiveExtGz    = ".gz"
	archiveExtXz    = ".xz"
	archiveExtZip   = ".zip"
	archiveExtTar   = ".tar"
	archiveExtBz2   = ".bz2"
	defaultDirMode  = 0755
	defaultFileMode = 0644
)

func (m *manager) extractArchive(ctx context.Context, archivePath string, extractDir string) (err error) {

	if err = os.MkdirAll(extractDir, defaultDirMode); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "extraction directory", err)
		return
	}

	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case archiveExtZip:
		if err = m.extractZip(ctx, archivePath, extractDir); err != nil {
			return
		}
		return
	case archiveExtGz:
		if strings.HasSuffix(strings.ToLower(archivePath), archiveExtTar+archiveExtGz) {
			if err = m.extractTarGz(ctx, archivePath, extractDir); err != nil {
				return
			}
			return
		}
		return fmt.Errorf(i18n.Msg("Unsupported archive format: %s"), ext)
	case archiveExtTar:
		if err = m.extractTar(ctx, archivePath, extractDir); err != nil {
			return
		}
		return
	default:
		return fmt.Errorf(i18n.Msg("Unsupported archive format: %s"), ext)
	}
}

// safeJoinPath предотвращает file traversal: проверяет relPath и baseDir перед Join.
func safeJoinPath(baseDir string, relPath string) (fullPath string, err error) {

	cleanPath := filepath.Clean(relPath)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf(i18n.Msg("Unsafe path: %s"), relPath)
	}

	fullPath = filepath.Join(baseDir, cleanPath)
	var absBase string
	if absBase, err = filepath.Abs(baseDir); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to get absolute path of base directory: %w"), err)
	}

	var absFull string
	if absFull, err = filepath.Abs(fullPath); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to get absolute path: %w"), err)
	}

	if !strings.HasPrefix(absFull, absBase) {
		return "", fmt.Errorf(i18n.Msg("Path is outside base directory: %s"), relPath)
	}

	return
}

func safeFileMode(mode int64) (fileMode os.FileMode) {

	if mode < 0 || mode > 0777 {
		return defaultFileMode
	}
	return os.FileMode(mode)
}

func (m *manager) extractZip(ctx context.Context, archivePath string, extractDir string) (err error) {

	var r *zip.ReadCloser
	if r, err = zip.OpenReader(archivePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to open ZIP archive: %w"), err)
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var path string
		if path, err = safeJoinPath(extractDir, f.Name); err != nil {
			return fmt.Errorf(i18n.Msg("Unsafe path in archive: %w"), err)
		}

		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(path, f.FileInfo().Mode()); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
		}

		var rc io.ReadCloser
		if rc, err = f.Open(); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to open file in archive: %w"), err)
		}

		var outFile *os.File
		if outFile, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode()); err != nil {
			_ = rc.Close()
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
		}

		//nolint:gosec // Ограничиваем размер файла для защиты от decompression bomb
		limitedReader := io.LimitReader(rc, maxFileSize)
		if _, err = io.Copy(outFile, limitedReader); err != nil {
			_ = rc.Close()
			_ = outFile.Close()
			return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
		}

		_ = rc.Close()
		_ = outFile.Close()
	}

	return
}

func (m *manager) extractTarGz(ctx context.Context, archivePath string, extractDir string) (err error) {

	var file *os.File
	if file, err = os.Open(archivePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to open archive: %w"), err)
	}
	defer func() { _ = file.Close() }()

	var gzr *gzip.Reader
	if gzr, err = gzip.NewReader(file); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "gzip reader", err)
	}
	defer func() { _ = gzr.Close() }()

	trArch := tar.NewReader(gzr)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var path string
		var header *tar.Header
		if header, err = trArch.Next(); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf(i18n.Msg("Error reading tar archive: %w"), err)
		}

		if path, err = safeJoinPath(extractDir, header.Name); err != nil {
			return fmt.Errorf(i18n.Msg("Unsafe path in archive: %w"), err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			//nolint:gosec // G703: path получен из safeJoinPath
			if err = os.MkdirAll(path, safeFileMode(header.Mode)); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			}
		case tar.TypeReg:
			//nolint:gosec // G703: path получен из safeJoinPath
			if err = os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			}

			var outFile *os.File
			//nolint:gosec // G703: path получен из safeJoinPath
			if outFile, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, safeFileMode(header.Mode)); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
			}

			//nolint:gosec // Ограничиваем размер файла для защиты от decompression bomb
			limitedReader := io.LimitReader(trArch, maxFileSize)
			if _, err = io.Copy(outFile, limitedReader); err != nil {
				_ = outFile.Close()
				return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
			}

			_ = outFile.Close()
		}
	}

	return
}

func (m *manager) extractTar(ctx context.Context, archivePath string, extractDir string) (err error) {

	var file *os.File
	if file, err = os.Open(archivePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to open archive: %w"), err)
	}
	defer func() { _ = file.Close() }()

	trArch := tar.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var path string
		var header *tar.Header
		if header, err = trArch.Next(); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf(i18n.Msg("Error reading tar archive: %w"), err)
		}
		if path, err = safeJoinPath(extractDir, header.Name); err != nil {
			return fmt.Errorf(i18n.Msg("Unsafe path in archive: %w"), err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			//nolint:gosec // G703: path получен из safeJoinPath
			if err = os.MkdirAll(path, safeFileMode(header.Mode)); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			}
		case tar.TypeReg:
			//nolint:gosec // G703: path получен из safeJoinPath
			if err = os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			}
			var outFile *os.File
			//nolint:gosec // G703: path получен из safeJoinPath
			if outFile, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, safeFileMode(header.Mode)); err != nil {
				return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
			}

			//nolint:gosec // Ограничиваем размер файла для защиты от decompression bomb
			limitedReader := io.LimitReader(trArch, maxFileSize)
			if _, err = io.Copy(outFile, limitedReader); err != nil {
				_ = outFile.Close()
				return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
			}
			_ = outFile.Close()
		}
	}

	return
}

func (m *manager) isArchive(filePath string) (isArchive bool) {

	ext := strings.ToLower(filepath.Ext(filePath))
	archiveExts := []string{archiveExtZip, archiveExtTar, archiveExtGz, archiveExtBz2, archiveExtXz}
	for _, archiveExt := range archiveExts {
		if ext == archiveExt || strings.HasSuffix(strings.ToLower(filePath), archiveExt+archiveExtGz) {
			return true
		}
	}
	return false
}
