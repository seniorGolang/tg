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

	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

const (
	protocolHTTP           = "http://"
	protocolHTTPS          = "https://"
	scriptTemplate         = "{{script}}"
	sourceTemplate         = "{{source}}"
	downloadedScriptPrefix = "downloaded_script_"
)

func (m *manager) executeScript(ctx context.Context, script *models.ScriptAction, workDir string, extractedDirs map[string]string) (err error) {

	if err = confirmScriptExecution(ctx, script); err != nil {
		return err
	}

	var cmd *exec.Cmd
	var scriptPath string

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
			return fmt.Errorf(i18n.Msg("Failed to resolve script source: %w"), err)
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
		return errors.New(i18n.Msg("Script not specified"))
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	var output []byte
	if output, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(i18n.Msg("Error executing script: %w, output: %s"), err, string(output))
	}

	return
}

func (m *manager) resolveScriptSource(ctx context.Context, source string, workDir string, extractedDirs map[string]string) (path string, err error) {

	if strings.HasPrefix(source, protocolHTTP) && !isForce(ctx) {
		// Риск: source=http://... загружает код, который затем исполняется на host.
		// Cleartext допускает MITM-подмену; без --force разрешаем только
		// HTTPS/file/package-local source. Это transport-level защита, не замена
		// подписи манифеста.
		return "", errors.New(i18n.Msg("HTTP script sources are not allowed without --force"))
	}

	if strings.HasPrefix(source, protocolHTTP) || strings.HasPrefix(source, protocolHTTPS) {
		scriptPath := filepath.Join(workDir, downloadedScriptPrefix+filepath.Base(source))
		if err = m.downloadScript(ctx, source, scriptPath); err != nil {
			return "", fmt.Errorf(i18n.Msg("Failed to download script: %w"), err)
		}
		path = scriptPath
		return
	}

	if strings.HasPrefix(source, protocolFile) {
		filePath := strings.TrimPrefix(source, protocolFile)
		var statErr error
		if _, statErr = os.Stat(filePath); statErr == nil {
			path = filePath
			return
		}
		return "", fmt.Errorf(i18n.Msg("Script not found: %s"), filePath)
	}

	for _, extractedDir := range extractedDirs {
		scriptPath := filepath.Join(extractedDir, source)
		var statErr error
		if _, statErr = os.Stat(scriptPath); statErr == nil {
			path = scriptPath
			return
		}
	}

	return "", fmt.Errorf(i18n.Msg("Script not found: %s"), source)
}

func confirmScriptExecution(ctx context.Context, script *models.ScriptAction) (err error) {

	if isForce(ctx) {
		return nil
	}

	message := i18n.Msg("Installer manifest requests host script execution. Review it carefully before continuing.")
	if script.Script != "" {
		message += "\n" + script.Script
	} else if script.Source != "" {
		message += "\n" + i18n.Msg("Script source: ") + script.Source
	}
	if script.Exec != "" {
		message += "\n" + i18n.Msg("Exec: ") + script.Exec
	}

	// Риск: pre_install/post_install запускаются на host с правами пользователя,
	// то есть это уже не WASM sandbox. Prompt делает trust boundary явным и показывает
	// inline/source/exec до запуска; --force остаётся осознанным bypass только для
	// доверенных манифестов и CI.
	var confirmed bool
	if confirmed, err = pterm.DefaultInteractiveConfirm.
		WithDefaultValue(false).
		Show(message); err != nil {
		return err
	}
	if !confirmed {
		return errors.New(i18n.Msg("script execution rejected by user"))
	}

	return nil
}

func isForce(ctx context.Context) (force bool) {

	if forceVal := ctx.Value(contextkeys.Force); forceVal != nil {
		if f, ok := forceVal.(bool); ok {
			return f
		}
	}
	return false
}

func (m *manager) downloadScript(ctx context.Context, url string, destPath string) (err error) {

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	client := &http.Client{}
	var resp *http.Response
	//nolint:gosec // G704: URL скрипта из манифеста/конфигурации
	if resp, err = client.Do(req); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to download script: %w"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(i18n.Msg("Unexpected status code: %d"), resp.StatusCode)
	}

	var file *os.File
	if file, err = os.Create(destPath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "file", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to write file: %w"), err)
	}

	return
}
