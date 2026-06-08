// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package command

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	wasmenv "github.com/seniorGolang/tg/v3/internal/wasm/env"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
	"github.com/seniorGolang/tg/v3/internal/wasm/stream"

	"github.com/goccy/go-json"
)

var commandResponses = struct {
	mu   sync.RWMutex
	data map[uint32]*CommandResponse
}{
	data: make(map[uint32]*CommandResponse),
}

const commandResponseRetention = 10 * time.Minute

// Выполняет команду на стороне хоста с валидацией по AllowedShellCMDs.
// commandPtr, commandLen - указатель и размер строки с командой (например, "go")
// argsPtr, argsLen - указатель и размер JSON массива строк с аргументами
// workDirPtr, workDirLen - указатель и размер строки с рабочей директорией (относительно rootDir)
// resultPtrPtr, resultSizePtr - указатели на указатель и размер результата (JSON с stdout, stderr, exitCode)
// Возвращает: 0 - успех, иначе код ошибки
func HostExecuteCommand(ctx context.Context, h *host.Host, commandPtr uint32, commandLen uint32, argsPtr uint32, argsLen uint32, workDirPtr uint32, workDirLen uint32, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	if h.Malloc == nil {
		return 1
	}

	var err error
	var args []string
	var workDir string
	var command string
	if command, args, workDir, err = readCommandInputs(h, commandPtr, commandLen, argsPtr, argsLen, workDirPtr, workDirLen); err != nil {
		resp := CommandResponse{Error: err.Error()}
		return memory.WriteObjectToPtrSize(ctx, h, resp, resultPtrPtr, resultSizePtr)
	}

	if err = validateCommand(h.Info, command); err != nil {
		resp := CommandResponse{Error: err.Error()}
		return memory.WriteObjectToPtrSize(ctx, h, resp, resultPtrPtr, resultSizePtr)
	}

	result := executeCommand(ctx, h, command, args, h.RootDir, workDir)

	return memory.WriteObjectToPtrSize(ctx, h, result, resultPtrPtr, resultSizePtr)
}

func readCommandInputs(h *host.Host, commandPtr uint32, commandLen uint32, argsPtr uint32, argsLen uint32, workDirPtr uint32, workDirLen uint32) (command string, args []string, workDir string, err error) {

	if command, err = memory.ReadString(h, commandPtr, commandLen); err != nil {
		return "", nil, "", fmt.Errorf(i18n.Msg("failed to read command: %w"), err)
	}

	if argsLen > 0 {
		argsJSONBytes, readErr := memory.ReadString(h, argsPtr, argsLen)
		if readErr != nil {
			return "", nil, "", fmt.Errorf(i18n.Msg("failed to read args: %w"), readErr)
		}
		if argsJSONBytes != "" {
			if err = json.Unmarshal([]byte(argsJSONBytes), &args); err != nil {
				return "", nil, "", fmt.Errorf(i18n.Msg("failed to decode args: %w"), err)
			}
		}
	}

	if workDir, err = memory.ReadString(h, workDirPtr, workDirLen); err != nil {
		return "", nil, "", fmt.Errorf(i18n.Msg("failed to read workDir: %w"), err)
	}
	if workDir == "" {
		workDir = "."
	}

	return
}

func executeCommand(ctx context.Context, h *host.Host, command string, args []string, rootDir string, workDir string) (response CommandResponse) {

	fullWorkDir, err := resolveCommandWorkDir(rootDir, workDir)
	if err != nil {
		return CommandResponse{
			Error: err.Error(),
		}
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = fullWorkDir
	// Риск: host-команда выполняется уже вне WASM sandbox, а os/exec по умолчанию
	// наследует окружение процесса, если Env оставить nil. os.Environ() здесь явно
	// отдал бы дочернему процессу все токены пользователя.
	// https://pkg.go.dev/os/exec#Cmd
	// Передаём только AllowedEnvVars, чтобы контракт capabilities совпадал с WASM env.
	// WebAssembly sandbox не защищает от возможностей, которые host сам экспортировал:
	// https://webassembly.org/docs/security/
	// См. OWASP OS Command Injection Defense Cheat Sheet:
	// https://cheatsheetseries.owasp.org/cheatsheets/OS_Command_Injection_Defense_Cheat_Sheet.html
	cmd.Env = buildCommandEnv(h.Info.AllowedEnvVars)

	cmdStdoutPipe, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		return CommandResponse{
			Error: fmt.Sprintf(i18n.Msg("Failed to create %s: %v"), "stdout pipe", stdoutErr),
		}
	}

	cmdStderrPipe, stderrErr := cmd.StderrPipe()
	if stderrErr != nil {
		cmdStdoutPipe.Close()
		return CommandResponse{
			Error: fmt.Sprintf(i18n.Msg("Failed to create %s: %v"), "stderr pipe", stderrErr),
		}
	}

	// Создаем потоки в реестре ДО запуска команды
	// Подключаем cmdStdoutPipe напрямую к StreamState.Reader
	// Это гарантирует, что потоки доступны плагину до того, как команда начнет выводить данные
	if h.StreamRegistry == nil {
		cmdStdoutPipe.Close()
		cmdStderrPipe.Close()
		return CommandResponse{
			Error: i18n.Msg("stream registry not available"),
		}
	}
	stdoutStreamID, stdoutErr := h.StreamRegistry.NewStream(ctx, h, cmdStdoutPipe, nil, stream.DefaultRingBufferSize)
	if stdoutErr != nil {
		cmdStdoutPipe.Close()
		cmdStderrPipe.Close()
		return CommandResponse{
			Error: fmt.Sprintf(i18n.Msg("Failed to create %s: %v"), "stdout stream", stdoutErr),
		}
	}

	stderrStreamID, stderrErr := h.StreamRegistry.NewStream(ctx, h, cmdStderrPipe, nil, stream.DefaultRingBufferSize)
	if stderrErr != nil {
		cmdStdoutPipe.Close()
		cmdStderrPipe.Close()
		h.StreamRegistry.CloseStream(ctx, h, stdoutStreamID)
		return CommandResponse{
			Error: fmt.Sprintf(i18n.Msg("Failed to create %s: %v"), "stderr stream", stderrErr),
		}
	}

	if err := cmd.Start(); err != nil {
		cmdStdoutPipe.Close()
		cmdStderrPipe.Close()
		h.StreamRegistry.CloseStream(ctx, h, stdoutStreamID)
		h.StreamRegistry.CloseStream(ctx, h, stderrStreamID)
		return CommandResponse{
			Error: fmt.Sprintf(i18n.Msg("failed to start command: %v"), err),
		}
	}

	stdoutStreamReader := stream.NewStreamReader(ctx, stdoutStreamID, h.StreamRegistry, h)
	stderrStreamReader := stream.NewStreamReader(ctx, stderrStreamID, h.StreamRegistry, h)

	stdoutStreamReader.StartReader()
	stderrStreamReader.StartReader()

	go func() {
		cmdErr := cmd.Wait()

		_ = cmdStdoutPipe.Close()
		_ = cmdStderrPipe.Close()

		h.StreamRegistry.WaitReaderDone(stdoutStreamID)
		h.StreamRegistry.WaitReaderDone(stderrStreamID)

		stdoutState, stdoutOk := h.StreamRegistry.GetStreamState(stdoutStreamID)
		stderrState, stderrOk := h.StreamRegistry.GetStreamState(stderrStreamID)
		if stdoutOk && stdoutState != nil {
			_ = stream.WaitForBufferEmpty(ctx, h, stdoutState.ReadBufferPtr, time.Millisecond*50, time.Second*5, time.Minute*10)
		}
		if stderrOk && stderrState != nil {
			_ = stream.WaitForBufferEmpty(ctx, h, stderrState.ReadBufferPtr, time.Millisecond*50, time.Second*5, time.Minute*10)
		}
		if stdoutOk && stdoutState != nil {
			_ = stream.SetClosed(ctx, h, stdoutState.ReadBufferPtr)
		}
		if stderrOk && stderrState != nil {
			_ = stream.SetClosed(ctx, h, stderrState.ReadBufferPtr)
		}

		commandResponses.mu.Lock()
		if resp, exists := commandResponses.data[stdoutStreamID]; exists {
			if cmdErr != nil {
				if exitErr, ok := cmdErr.(*exec.ExitError); ok {
					resp.ExitCode = exitErr.ExitCode()
				} else {
					h.StreamRegistry.CloseStream(ctx, h, stdoutStreamID)
					h.StreamRegistry.CloseStream(ctx, h, stderrStreamID)
					resp.ExitCode = -1
				}
			} else {
				resp.ExitCode = 0
			}
		}
		commandResponses.mu.Unlock()

		scheduleCommandResponseCleanup(stdoutStreamID)
	}()

	stdoutStreamIDInt32 := int32(stdoutStreamID) //nolint:gosec // streamID всегда < MaxInt32
	stderrStreamIDInt32 := int32(stderrStreamID) //nolint:gosec // streamID всегда < MaxInt32

	response = CommandResponse{
		ExitCode:       -2,
		StdoutStreamID: stdoutStreamIDInt32,
		StderrStreamID: stderrStreamIDInt32,
	}

	commandResponses.mu.Lock()
	commandResponses.data[stdoutStreamID] = &response
	commandResponses.mu.Unlock()

	return
}

func resolveCommandWorkDir(rootDir string, workDir string) (fullWorkDir string, err error) {

	if workDir == "" {
		workDir = "."
	}

	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return "", fmt.Errorf(i18n.Msg("failed to resolve root directory: %w"), err)
	}

	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, workDir))
	if err != nil {
		return "", fmt.Errorf(i18n.Msg("failed to resolve workDir: %w"), err)
	}

	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", fmt.Errorf(i18n.Msg("failed to validate workDir: %w"), err)
	}

	// Риск: workDir приходит из WASM и filepath.Join сам по себе не является
	// containment-check: после Clean путь вроде ../../ может выйти за rootDir.
	// Проверяем filepath.Rel и запрещаем path traversal, чтобы разрешённая
	// shell-команда не обходила FS sandbox через рабочую директорию.
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf(i18n.Msg("workDir %s escapes plugin root"), workDir)
	}

	return targetAbs, nil
}

func buildCommandEnv(allowedEnvVars []string) (out []string) {

	envVars := wasmenv.ResolveEnvVars(allowedEnvVars)
	out = make([]string, 0, len(envVars)+1)
	for key, value := range envVars {
		out = append(out, key+"="+value)
	}

	out = append(out, "TG_LANG="+i18n.GetLanguage())
	return out
}

func scheduleCommandResponseCleanup(streamID uint32) {

	// Страховка от утечки памяти, если плагин завершил команду, но не вызвал
	// host_get_command_response для чтения финального результата.
	time.AfterFunc(commandResponseRetention, func() {
		commandResponses.mu.Lock()
		delete(commandResponses.data, streamID)
		commandResponses.mu.Unlock()
	})
}

func validateCommand(info plugin.Info, command string) (err error) {

	if len(info.AllowedShellCMDs) == 0 {
		err = fmt.Errorf(i18n.Msg("command %s is not allowed: plugin has no allowed commands"), command)
		return
	}

	for _, allowed := range info.AllowedShellCMDs {
		if allowed == command {
			return
		}
	}

	return fmt.Errorf(i18n.Msg("command %s is not allowed: not found in allowed commands list"), command)
}

// HostGetStreamReadBufferPtr: streamID — ID потока.
// bufferPtrPtr - указатель на uint32, куда будет записан указатель на буфер
// Возвращает: 0 - успех, иначе код ошибки
func HostGetStreamReadBufferPtr(ctx context.Context, h *host.Host, streamID uint32, bufferPtrPtr uint32) (resultCode uint32) {

	if h.StreamRegistry == nil {
		return 1
	}

	bufferPtr, ok := h.StreamRegistry.GetReadBufferPtr(streamID)
	if !ok || bufferPtr == 0 {
		return 1
	}

	ptrBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ptrBytes, bufferPtr)

	if err := memory.Write(h, bufferPtrPtr, ptrBytes); err != nil {
		return 1
	}

	return 0
}

// HostGetCommandResponse: streamID — ID потока stdout команды.
// resultPtrPtr, resultSizePtr - указатели на указатель и размер результата (JSON)
// Возвращает: 0 - успех, иначе код ошибки
func HostGetCommandResponse(ctx context.Context, h *host.Host, streamID uint32, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	resp, exists := getCommandResponse(streamID)

	if !exists {
		resp = &CommandResponse{ExitCode: -1}
	}

	return memory.WriteObjectToPtrSize(ctx, h, resp, resultPtrPtr, resultSizePtr)
}

func getCommandResponse(streamID uint32) (resp *CommandResponse, exists bool) {

	commandResponses.mu.Lock()
	defer commandResponses.mu.Unlock()

	resp, exists = commandResponses.data[streamID]
	if exists && resp != nil && resp.ExitCode != -2 {
		// Риск: без удаления глобальная map растёт при каждом запуске команды.
		// Удаляем только финальный результат; промежуточный ExitCode=-2 нужен
		// polling-клиенту как сигнал "команда ещё выполняется".
		delete(commandResponses.data, streamID)
	}
	return
}
