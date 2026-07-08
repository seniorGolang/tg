// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/ui"
)

const (
	// executableFileMode права доступа для исполняемых файлов
	executableFileMode = 0700
	// executableBitMask маска для проверки исполняемости файла
	executableBitMask = 0111
	// binPath путь к бинарным файлам
	binPath = "/bin/"
)

// copyFileWithProgress копирует файл из источника в назначение с отображением прогресса.
func (m *manager) copyFileWithProgress(sourcePath string, destination string, sourceFile string, progressBar *ui.ProgressBar) (err error) {

	var actualSourcePath string
	if sourceFile != "" {
		actualSourcePath = filepath.Join(filepath.Dir(sourcePath), sourceFile)
		var statErr error
		if _, statErr = os.Stat(actualSourcePath); statErr != nil {
			actualSourcePath = sourcePath
		}
	} else {
		actualSourcePath = sourcePath
	}

	var sourceFileHandle *os.File
	if sourceFileHandle, err = os.Open(actualSourcePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to read source file: %w"), err)
	}
	defer func() { _ = sourceFileHandle.Close() }()

	var sourceInfo os.FileInfo
	if sourceInfo, err = sourceFileHandle.Stat(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get source file info: %w"), err)
	}
	totalSize := sourceInfo.Size()

	var destFileHandle *os.File
	if destFileHandle, err = os.Create(destination); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to write file: %w"), err)
	}
	defer func() { _ = destFileHandle.Close() }()

	var reader io.Reader = sourceFileHandle
	if progressBar != nil && totalSize > 0 {
		reader = &copyProgressReader{
			reader: sourceFileHandle,
			total:  totalSize,
			copied: 0,
			bar:    progressBar,
		}
	}

	if _, err = io.Copy(destFileHandle, reader); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to copy file: %w"), err)
	}

	if progressBar != nil && totalSize > 0 {
		progressBar.SetCurrent(100)
		progressBar.Print()
	}

	if m.isExecutable(destination) {
		if err = os.Chmod(destination, executableFileMode); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to set file permissions: %w"), err)
		}
	}

	return
}

// copyProgressReader оборачивает io.Reader и отслеживает прогресс копирования для отображения в прогресс-баре.
type copyProgressReader struct {
	reader      io.Reader
	total       int64
	copied      int64
	bar         *ui.ProgressBar
	lastPercent int // Последний отображенный процент для избежания лишних обновлений
}

func (cpr *copyProgressReader) Read(p []byte) (n int, err error) {

	if n, err = cpr.reader.Read(p); n > 0 {
		cpr.copied += int64(n)

		if cpr.bar != nil && cpr.total > 0 {
			percentage := int(float64(cpr.copied) / float64(cpr.total) * 100)
			if percentage > 100 {
				percentage = 100
			}
			if percentage != cpr.lastPercent {
				cpr.lastPercent = percentage
				cpr.bar.SetCurrent(percentage)
				cpr.bar.Print()
			}
		}
	}
	return
}

func (m *manager) isExecutable(filePath string) (isExecutable bool) {

	if strings.HasSuffix(filePath, plugin.FileExtTGP) {
		return true
	}

	if strings.Contains(filePath, binPath) {
		return true
	}

	var err error
	var info os.FileInfo
	if info, err = os.Stat(filePath); err != nil {
		return false
	}

	return info.Mode()&executableBitMask != 0
}
