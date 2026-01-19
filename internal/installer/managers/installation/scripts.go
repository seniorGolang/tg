// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

const (
	protocolHTTP           = "http://"
	protocolHTTPS          = "https://"
	scriptTemplate         = "{{script}}"
	sourceTemplate         = "{{source}}"
	downloadedScriptPrefix = "downloaded_script_"
)

// executeScript выполняет скрипт.
func (m *manager) executeScript(ctx context.Context, script *models.ScriptAction, workDir string, extractedDirs map[string]string) (err error) {

	var scriptPath string
	var cmd *exec.Cmd

	switch {
	case script.Script != "":
		execCmd := script.Exec
		if strings.Contains(execCmd, scriptTemplate) {
			execCmd = strings.ReplaceAll(execCmd, scriptTemplate, script.Script)
			parts := strings.Fields(execCmd)
			//nolint:gosec // Команда и аргументы приходят из доверенного манифеста
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		} else {
			execParts := strings.Fields(execCmd)
			args := make([]string, 0, len(execParts)+2)
			args = append(args, execParts[1:]...)
			args = append(args, "-c", script.Script)
			//nolint:gosec // Команда и аргументы приходят из доверенного манифеста
			cmd = exec.CommandContext(ctx, execParts[0], args...)
		}
	case script.Source != "":
		if scriptPath, err = m.resolveScriptSource(ctx, script.Source, workDir, extractedDirs); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to resolve script source: %w"), err)
			return
		}

		execCmd := script.Exec
		if strings.Contains(execCmd, sourceTemplate) {
			execCmd = strings.ReplaceAll(execCmd, sourceTemplate, scriptPath)
			parts := strings.Fields(execCmd)
			//nolint:gosec // Команда и аргументы приходят из доверенного манифеста
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		} else {
			execParts := strings.Fields(execCmd)
			args := make([]string, 0, len(execParts)+1)
			args = append(args, execParts[1:]...)
			args = append(args, scriptPath)
			//nolint:gosec // Команда и аргументы приходят из доверенного манифеста
			cmd = exec.CommandContext(ctx, execParts[0], args...)
		}
	default:
		err = errors.New(i18n.Msg("Script not specified"))
		return
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	var output []byte
	if output, err = cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf(i18n.Msg("Error executing script: %w, output: %s"), err, string(output))
		return
	}

	return
}

// resolveScriptSource разрешает источник скрипта.
func (m *manager) resolveScriptSource(ctx context.Context, source string, workDir string, extractedDirs map[string]string) (path string, err error) {

	if strings.HasPrefix(source, protocolHTTP) || strings.HasPrefix(source, protocolHTTPS) {
		scriptPath := filepath.Join(workDir, downloadedScriptPrefix+filepath.Base(source))
		if err = m.downloadScript(ctx, source, scriptPath); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to download script: %w"), err)
			return
		}
		path = scriptPath
		return
	}

	if strings.HasPrefix(source, protocolFile) {
		// Убираем префикс file:// и используем путь напрямую
		filePath := strings.TrimPrefix(source, protocolFile)
		var statErr error
		if _, statErr = os.Stat(filePath); statErr == nil {
			path = filePath
			return
		}
		err = fmt.Errorf(i18n.Msg("Script not found: %s"), filePath)
		return
	}

	for _, extractedDir := range extractedDirs {
		scriptPath := filepath.Join(extractedDir, source)
		var statErr error
		if _, statErr = os.Stat(scriptPath); statErr == nil {
			path = scriptPath
			return
		}
	}

	err = fmt.Errorf(i18n.Msg("Script not found: %s"), source)
	return
}

// downloadScript загружает скрипт по URL.
func (m *manager) downloadScript(ctx context.Context, url string, destPath string) (err error) {

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
		return
	}

	client := &http.Client{}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to download script: %w"), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(i18n.Msg("Unexpected status code: %d"), resp.StatusCode)
		return
	}

	var file *os.File
	if file, err = os.Create(destPath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
		return
	}
	defer file.Close()

	if _, err = io.Copy(file, resp.Body); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to write file: %w"), err)
		return
	}

	return
}
