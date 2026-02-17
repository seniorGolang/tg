// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type CoreCreator struct{}

func (c *CoreCreator) Create(rootDir string, moduleName string) (err error) {

	// В WASM файловая система монтируется в корень "/", поэтому используем относительные пути
	coreDir := CoreDirName
	if err = os.MkdirAll(coreDir, 0755); err != nil {
		return
	}

	wasmDir := filepath.Join(coreDir, CoreWasmSubDir)
	if err = os.MkdirAll(wasmDir, 0755); err != nil {
		return
	}

	execDir := filepath.Join(coreDir, CoreExecSubDir)
	if err = os.MkdirAll(execDir, 0755); err != nil {
		return
	}

	httpDir := filepath.Join(coreDir, CoreHTTPSubDir)
	if err = os.MkdirAll(httpDir, 0755); err != nil {
		return
	}

	dataDir := filepath.Join(coreDir, CoreDataSubDir)
	if err = os.MkdirAll(dataDir, 0755); err != nil {
		return
	}

	pluginDir := filepath.Join(coreDir, CorePluginSubDir)
	if err = os.MkdirAll(pluginDir, 0755); err != nil {
		return
	}

	netDir := filepath.Join(coreDir, CoreNetSubDir)
	if err = os.MkdirAll(netDir, 0755); err != nil {
		return
	}

	i18nDir := filepath.Join(coreDir, "i18n")
	if err = os.MkdirAll(i18nDir, 0755); err != nil {
		return
	}

	manifestDir := filepath.Join(coreDir, "manifest")
	if err = os.MkdirAll(manifestDir, 0755); err != nil {
		return
	}

	data := TemplateData{
		ModuleName: moduleName,
	}

	// Файлы в корне core/
	// В tests/plugin/core в корне только 4 файла: init.go, init_wasm.go, interactive.go, task.go, generator.go
	coreFiles := map[string]string{
		"init.go":        "templates/core_init.go.tmpl",
		"init_wasm.go":   "templates/core_init_wasm.go.tmpl",
		"interactive.go": "templates/core_interactive.go.tmpl",
		"task.go":        "templates/core_task.go.tmpl",
		"generator.go":   "templates/core_generator.go.tmpl",
	}

	// Файлы в core/wasm/
	// В tests/plugin/core/wasm есть: command.go, connection.go, errors.go, exchange.go, execute.go, export.go,
	// host.go, info.go, interactive_wasm.go, interactive.go, logger_wasm.go, logger.go, memory.go,
	// plugin_init.go, polling.go, result.go, ringbuffer.go, task_default.go, task.go, generator.go, generator_default.go
	wasmFiles := map[string]string{
		"command.go":           "templates/wasm_command.go.tmpl",
		"connection.go":        "templates/wasm_connection.go.tmpl",
		"errors.go":            "templates/wasm_errors.go.tmpl",
		"exchange.go":          "templates/wasm_exchange.go.tmpl",
		"execute.go":           "templates/wasm_execute.go.tmpl",
		"export.go":            "templates/wasm_export.go.tmpl",
		"generator.go":         "templates/wasm_generator.go.tmpl",
		"generator_default.go": "templates/wasm_generator_default.go.tmpl",
		"host.go":              "templates/wasm_host.go.tmpl",
		"info.go":              "templates/wasm_info.go.tmpl",
		"interactive.go":       "templates/wasm_interactive_select.go.tmpl",
		"interactive_wasm.go":  "templates/wasm_interactive_wasip1.go.tmpl",
		"logger.go":            "templates/wasm_logger.go.tmpl",
		"logger_wasm.go":       "templates/wasm_logger_wasm.go.tmpl",
		"memory.go":            "templates/wasm_memory.go.tmpl",
		"plugin_init.go":       "templates/wasm_plugin_init.go.tmpl",
		"polling.go":           "templates/wasm_polling.go.tmpl",
		"result.go":            "templates/wasm_result.go.tmpl",
		"ringbuffer.go":        "templates/wasm_ringbuffer.go.tmpl",
		"task.go":              "templates/wasm_task.go.tmpl",
		"task_default.go":      "templates/wasm_task_default.go.tmpl",
	}

	// Файлы в core/exec/
	execFiles := map[string]string{
		"exec.go":      "templates/core_exec_exec.go.tmpl",
		"exec_wasm.go": "templates/core_exec_exec_wasm.go.tmpl",
	}

	// Файлы в core/http/
	httpFiles := map[string]string{
		"aliases.go":              "templates/core_http_aliases.go.tmpl",
		"server.go":               "templates/core_http_server.go.tmpl",
		"server_listen.go":        "templates/core_http_server_listen.go.tmpl",
		"server_listen_wasm.go":   "templates/core_http_server_listen_wasm.go.tmpl",
		"server_response_wasm.go": "templates/core_http_server_response_writer_wasm.go.tmpl",
		"server_stop.go":          "templates/core_http_server_stop.go.tmpl",
		"server_stop_wasm.go":     "templates/core_http_server_stop_wasm.go.tmpl",
		"dispatch_wasm.go":        "templates/core_http_dispatch_wasm.go.tmpl",
		"host_wasm.go":            "templates/core_http_host_wasm.go.tmpl",
		"client.go":               "templates/core_http_client.go.tmpl",
		"client_wasm.go":          "templates/core_http_client_wasm.go.tmpl",
	}

	// Файлы в core/data/
	dataFiles := map[string]string{
		"storage.go": "templates/core_data_storage.go.tmpl",
	}

	// Файлы в core/plugin/
	pluginFiles := map[string]string{
		"plugin.go": "templates/core_plugin_plugin.go.tmpl",
		"info.go":   "templates/core_plugin_info.go.tmpl",
	}

	// Файлы в core/net/
	netFiles := map[string]string{
		"conn.go":          "templates/core_net_conn.go.tmpl",
		"conn_addr.go":     "templates/core_net_conn_addr.go.tmpl",
		"conn_deadline.go": "templates/core_net_conn_deadline.go.tmpl",
		"conn_dial.go":     "templates/core_net_conn_dial.go.tmpl",
		"conn_factory.go":  "templates/core_net_conn_factory.go.tmpl",
		"conn_io.go":       "templates/core_net_conn_io.go.tmpl",
		"constants.go":     "templates/core_net_constants.go.tmpl",
		"handler.go":       "templates/core_net_handler.go.tmpl",
		"listener.go":      "templates/core_net_listener.go.tmpl",
		"listener_addr.go": "templates/core_net_listener_addr.go.tmpl",
		"listener_stop.go": "templates/core_net_listener_stop.go.tmpl",
		"wasi_import.go":   "templates/core_net_wasi_import.go.tmpl",
	}

	// Файлы в core/i18n/
	i18nFiles := map[string]string{
		"msg.go":    "templates/core_i18n_msg.go.tmpl",
		"detect.go": "templates/core_i18n_detect.go.tmpl",
		"load.go":   "templates/core_i18n_load.go.tmpl",
	}

	// Файлы в core/manifest/
	manifestFiles := map[string]string{
		"types.go":    "templates/core_manifest_types.go.tmpl",
		"generate.go": "templates/core_manifest_generate.go.tmpl",
		"args.go":     "templates/core_manifest_args.go.tmpl",
	}

	// Генерируем файлы в корне core/
	for filename, templatePath := range coreFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(coreDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), filename, err)
		}
	}

	// Генерируем файлы в core/wasm/
	for filename, templatePath := range wasmFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(wasmDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "wasm/"+filename, err)
		}
	}

	// Генерируем файлы в core/exec/
	for filename, templatePath := range execFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(execDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "exec/"+filename, err)
		}
	}

	// Генерируем файлы в core/http/
	for filename, templatePath := range httpFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(httpDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "http/"+filename, err)
		}
	}

	// Генерируем файлы в core/data/
	for filename, templatePath := range dataFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(dataDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "data/"+filename, err)
		}
	}

	// Генерируем файлы в core/plugin/
	for filename, templatePath := range pluginFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(pluginDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "plugin/"+filename, err)
		}
	}

	// Генерируем файлы в core/net/
	for filename, templatePath := range netFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(netDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "net/"+filename, err)
		}
	}

	// Генерируем файлы в core/i18n/
	// В core/i18n/ только Go файлы, файлы переводов находятся в корневом пакете i18n/
	for filename, templatePath := range i18nFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(i18nDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "i18n/"+filename, err)
		}
	}

	// Генерируем файлы в core/manifest/
	for filename, templatePath := range manifestFiles {
		var content string
		if content, err = renderTemplate(templatePath, data); err != nil {
			return fmt.Errorf(i18n.Msg("failed to render %s: %w"), templatePath, err)
		}

		filePath := filepath.Clean(filepath.Join(manifestDir, filename))
		if err = writeFile(filePath, content); err != nil {
			return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "manifest/"+filename, err)
		}
	}

	return
}

func (c *CoreCreator) Exists() (exists bool) {

	coreDir := CoreDirName
	var err error
	if _, err = os.Stat(coreDir); err == nil {
		exists = true
		return
	}

	return
}
