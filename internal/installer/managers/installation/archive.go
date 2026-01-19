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
	// maxFileSize ограничивает максимальный размер файла при распаковке (100MB)
	maxFileSize = 100 * 1024 * 1024
	// defaultDirMode права доступа для директорий по умолчанию
	defaultDirMode = 0755
	// defaultFileMode права доступа для файлов по умолчанию
	defaultFileMode = 0644
	// archiveExtZip расширение ZIP архива
	archiveExtZip = ".zip"
	// archiveExtTar расширение TAR архива
	archiveExtTar = ".tar"
	// archiveExtGz расширение GZ архива
	archiveExtGz = ".gz"
	// archiveExtBz2 расширение BZ2 архива
	archiveExtBz2 = ".bz2"
	// archiveExtXz расширение XZ архива
	archiveExtXz = ".xz"
)

// extractArchive распаковывает архив во временную директорию.
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
		err = fmt.Errorf(i18n.Msg("Unsupported archive format: %s"), ext)
		return
	case archiveExtTar:
		if err = m.extractTar(ctx, archivePath, extractDir); err != nil {
			return
		}
		return
	default:
		err = fmt.Errorf(i18n.Msg("Unsupported archive format: %s"), ext)
		return
	}
}

// safeJoinPath безопасно объединяет пути, предотвращая file traversal атаки.
func safeJoinPath(baseDir string, relPath string) (fullPath string, err error) {

	cleanPath := filepath.Clean(relPath)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		err = fmt.Errorf(i18n.Msg("Unsafe path: %s"), relPath)
		return
	}

	fullPath = filepath.Join(baseDir, cleanPath)
	var absBase string
	if absBase, err = filepath.Abs(baseDir); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get absolute path of base directory: %w"), err)
		return
	}

	var absFull string
	if absFull, err = filepath.Abs(fullPath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get absolute path: %w"), err)
		return
	}

	if !strings.HasPrefix(absFull, absBase) {
		err = fmt.Errorf(i18n.Msg("Path is outside base directory: %s"), relPath)
		return
	}

	return
}

// safeFileMode безопасно преобразует int64 в os.FileMode.
func safeFileMode(mode int64) (fileMode os.FileMode) {

	if mode < 0 || mode > 0777 {
		fileMode = defaultFileMode
		return
	}

	fileMode = os.FileMode(mode)
	return
}

// extractZip распаковывает ZIP архив.
func (m *manager) extractZip(ctx context.Context, archivePath string, extractDir string) (err error) {

	var r *zip.ReadCloser
	if r, err = zip.OpenReader(archivePath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to open ZIP archive: %w"), err)
		return
	}
	defer r.Close()

	for _, f := range r.File {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var path string
		if path, err = safeJoinPath(extractDir, f.Name); err != nil {
			err = fmt.Errorf(i18n.Msg("Unsafe path in archive: %w"), err)
			return
		}

		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(path, f.FileInfo().Mode()); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
				return
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
			return
		}

		var rc io.ReadCloser
		if rc, err = f.Open(); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to open file in archive: %w"), err)
			return
		}

		var outFile *os.File
		if outFile, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode()); err != nil {
			rc.Close()
			err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
			return
		}

		//nolint:gosec // Ограничиваем размер файла для защиты от decompression bomb
		limitedReader := io.LimitReader(rc, maxFileSize)
		if _, err = io.Copy(outFile, limitedReader); err != nil {
			rc.Close()
			outFile.Close()
			err = fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
			return
		}

		rc.Close()
		outFile.Close()
	}

	return
}

// extractTarGz распаковывает tar.gz архив.
func (m *manager) extractTarGz(ctx context.Context, archivePath string, extractDir string) (err error) {

	var file *os.File
	if file, err = os.Open(archivePath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to open archive: %w"), err)
		return
	}
	defer file.Close()

	var gzr *gzip.Reader
	if gzr, err = gzip.NewReader(file); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "gzip reader", err)
		return
	}
	defer gzr.Close()

	trArch := tar.NewReader(gzr)
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var header *tar.Header
		if header, err = trArch.Next(); err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf(i18n.Msg("Error reading tar archive: %w"), err)
			return
		}

		var path string
		if path, err = safeJoinPath(extractDir, header.Name); err != nil {
			err = fmt.Errorf(i18n.Msg("Unsafe path in archive: %w"), err)
			return
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, safeFileMode(header.Mode)); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
				return
			}
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
				return
			}

			var outFile *os.File
			if outFile, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, safeFileMode(header.Mode)); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
				return
			}

			//nolint:gosec // Ограничиваем размер файла для защиты от decompression bomb
			limitedReader := io.LimitReader(trArch, maxFileSize)
			if _, err = io.Copy(outFile, limitedReader); err != nil {
				outFile.Close()
				err = fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
				return
			}

			outFile.Close()
		}
	}

	return
}

// extractTar распаковывает tar архив.
func (m *manager) extractTar(ctx context.Context, archivePath string, extractDir string) (err error) {

	var file *os.File
	if file, err = os.Open(archivePath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to open archive: %w"), err)
		return
	}
	defer file.Close()

	trArch := tar.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var header *tar.Header
		if header, err = trArch.Next(); err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf(i18n.Msg("Error reading tar archive: %w"), err)
			return
		}
		var path string
		if path, err = safeJoinPath(extractDir, header.Name); err != nil {
			err = fmt.Errorf(i18n.Msg("Unsafe path in archive: %w"), err)
			return
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, safeFileMode(header.Mode)); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
				return
			}
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
				return
			}
			var outFile *os.File
			if outFile, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, safeFileMode(header.Mode)); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
				return
			}

			//nolint:gosec // Ограничиваем размер файла для защиты от decompression bomb
			limitedReader := io.LimitReader(trArch, maxFileSize)
			if _, err = io.Copy(outFile, limitedReader); err != nil {
				_ = outFile.Close()
				err = fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
				return
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
