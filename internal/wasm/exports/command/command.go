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
	"sync"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
	"github.com/seniorGolang/tg/v3/internal/wasm/stream"

	"github.com/goccy/go-json"
)

// commandResponses хранит CommandResponse для каждой команды по stdoutStreamID.
// Используется для получения реального exitCode после завершения команды.
var commandResponses = struct {
	mu   sync.RWMutex
	data map[uint32]*CommandResponse
}{
	data: make(map[uint32]*CommandResponse),
}

// HostExecuteCommand обрабатывает вызов host_execute_command из плагина.
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

	// Читаем входные данные
	command, args, workDir, err := readCommandInputs(h, commandPtr, commandLen, argsPtr, argsLen, workDirPtr, workDirLen)
	if err != nil {
		result := CommandResponse{Error: err.Error()}
		return memory.WriteObjectToPtrSize(ctx, h, result, resultPtrPtr, resultSizePtr)
	}

	if err = validateCommand(h.Info, command); err != nil {
		result := CommandResponse{Error: err.Error()}
		return memory.WriteObjectToPtrSize(ctx, h, result, resultPtrPtr, resultSizePtr)
	}

	result := executeCommand(ctx, h, command, args, h.RootDir, workDir)

	return memory.WriteObjectToPtrSize(ctx, h, result, resultPtrPtr, resultSizePtr)
}

func readCommandInputs(h *host.Host, commandPtr uint32, commandLen uint32, argsPtr uint32, argsLen uint32, workDirPtr uint32, workDirLen uint32) (command string, args []string, workDir string, err error) {

	command, err = memory.ReadString(h, commandPtr, commandLen)
	if err != nil {
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

	workDir, err = memory.ReadString(h, workDirPtr, workDirLen)
	if err != nil {
		return "", nil, "", fmt.Errorf(i18n.Msg("failed to read workDir: %w"), err)
	}
	if workDir == "" {
		workDir = "."
	}

	return
}

// executeCommand выполняет команду и возвращает результат.
// Использует стриминг для stdout/stderr через cmd.StdoutPipe/StderrPipe, чтобы не загружать весь вывод в память.
func executeCommand(ctx context.Context, h *host.Host, command string, args []string, rootDir string, workDir string) (response CommandResponse) {

	fullWorkDir := filepath.Join(rootDir, workDir)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = fullWorkDir
	cmd.Env = os.Environ()

	// Получаем pipe для stdout и stderr ДО запуска команды
	// Это позволяет читать данные в реальном времени
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
	// Используем размер буфера по умолчанию для кольцевых буферов
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

	// Запускаем команду асинхронно
	if err := cmd.Start(); err != nil {
		cmdStdoutPipe.Close()
		cmdStderrPipe.Close()
		h.StreamRegistry.CloseStream(ctx, h, stdoutStreamID)
		h.StreamRegistry.CloseStream(ctx, h, stderrStreamID)
		return CommandResponse{
			Error: fmt.Sprintf(i18n.Msg("failed to start command: %v"), err),
		}
	}

	// Создаем StreamReader для чтения данных из pipe и записи в кольцевые буферы
	streamReg, ok := h.StreamRegistry.(*stream.StreamRegistry)
	if !ok || streamReg == nil {
		cmdStdoutPipe.Close()
		cmdStderrPipe.Close()
		h.StreamRegistry.CloseStream(ctx, h, stdoutStreamID)
		h.StreamRegistry.CloseStream(ctx, h, stderrStreamID)
		return CommandResponse{
			Error: i18n.Msg("stream registry type mismatch"),
		}
	}
	stdoutStreamReader := stream.NewStreamReader(ctx, stdoutStreamID, streamReg, h)
	stderrStreamReader := stream.NewStreamReader(ctx, stderrStreamID, streamReg, h)

	// Запускаем горутины для чтения данных из pipe и записи в кольцевые буферы
	// StreamReader.StartReader() запускает горутину, которая читает из Reader (cmdStdoutPipe) и записывает в кольцевой буфер
	stdoutStreamReader.StartReader()
	stderrStreamReader.StartReader()

	// Не ждем завершения команды здесь, чтобы избежать deadlock при заполненном буфере.
	// Завершение команды отслеживается в фоне, exitCode обновляется после завершения.

	// Запускаем горутину для отслеживания завершения команды в фоне
	// StartReader() читает данные из pipe и записывает в кольцевой буфер
	// Когда pipe закроется (команда завершится), StartReader() получит EOF и установит state.Reader = nil
	// Плагин получит EOF при чтении из кольцевого буфера, когда все данные будут прочитаны
	go func() {
		cmdErr := cmd.Wait()

		// Закрываем pipe после завершения команды
		// Это сигнализирует StartReader(), что данных больше не будет
		_ = cmdStdoutPipe.Close()
		_ = cmdStderrPipe.Close()

		// Ждем завершения StartReader() перед установкой флага Closed.
		h.StreamRegistry.WaitReaderDone(stdoutStreamID)
		h.StreamRegistry.WaitReaderDone(stderrStreamID)

		// Ждем, пока все данные будут прочитаны плагином из кольцевого буфера.
		// WaitForBufferEmpty отслеживает изменение ReadIndex для обнаружения зависших плагинов.
		if stdoutState := h.StreamRegistry.GetStreamState(stdoutStreamID); stdoutState != nil {
			if state, ok := stdoutState.(*stream.StreamState); ok && state != nil {
				_ = stream.WaitForBufferEmpty(
					ctx,
					h,
					state.ReadBufferPtr,
					time.Millisecond*50,
					time.Second*5,
					time.Minute*10,
				)
			}
		}

		// Аналогично для stderr
		if stderrState := h.StreamRegistry.GetStreamState(stderrStreamID); stderrState != nil {
			if state, ok := stderrState.(*stream.StreamState); ok && state != nil {
				_ = stream.WaitForBufferEmpty(
					ctx,
					h,
					state.ReadBufferPtr,
					time.Millisecond*50,
					time.Second*5,
					time.Minute*10,
				)
			}
		}

		// ТЕПЕРЬ устанавливаем флаг Closed, когда все данные прочитаны плагином
		// Все данные уже записаны в буфер и прочитаны плагином
		if stdoutState := h.StreamRegistry.GetStreamState(stdoutStreamID); stdoutState != nil {
			if state, ok := stdoutState.(*stream.StreamState); ok && state != nil {
				_ = stream.SetClosed(ctx, h, state.ReadBufferPtr)
			}
		}
		if stderrState := h.StreamRegistry.GetStreamState(stderrStreamID); stderrState != nil {
			if state, ok := stderrState.(*stream.StreamState); ok && state != nil {
				_ = stream.SetClosed(ctx, h, state.ReadBufferPtr)
			}
		}

		// Обновляем CommandResponse с реальным exitCode после завершения команды
		commandResponses.mu.Lock()
		if resp, exists := commandResponses.data[stdoutStreamID]; exists {
			if cmdErr != nil {
				if exitErr, ok := cmdErr.(*exec.ExitError); ok {
					// Команда завершилась с ненулевым кодом выхода
					resp.ExitCode = exitErr.ExitCode()
				} else {
					// Другая ошибка (не exit error) - закрываем потоки
					h.StreamRegistry.CloseStream(ctx, h, stdoutStreamID)
					h.StreamRegistry.CloseStream(ctx, h, stderrStreamID)
					// Для других ошибок устанавливаем exitCode = -1
					resp.ExitCode = -1
				}
			} else {
				// Команда завершилась успешно
				resp.ExitCode = 0
			}
		}
		commandResponses.mu.Unlock()
	}()

	// НЕ ждем завершения копирования данных - это может привести к deadlock,
	// если буфер заполнен, а плагин еще не начал читать
	// Копирование продолжится в фоне, а плагин начнет читать данные порциями
	// wgCopy.Wait() - убрано для предотвращения deadlock

	// Возвращаем streamID сразу, не дожидаясь завершения команды
	// Это позволяет плагину начать читать данные, что разблокирует writer
	// Всегда возвращаем streamID, даже если данных нет
	// Плагин должен проверить наличие данных при чтении
	// streamID всегда положительный и меньше MaxInt32, безопасное преобразование uint32 -> int32
	stdoutStreamIDInt32 := int32(stdoutStreamID) //nolint:gosec // streamID всегда < MaxInt32
	stderrStreamIDInt32 := int32(stderrStreamID) //nolint:gosec // streamID всегда < MaxInt32

	// Создаем CommandResponse и сохраняем в хранилище для последующего обновления exitCode
	// Используем -2 как специальное значение для "команда еще не завершена"
	// Это позволяет отличить незавершенную команду от успешного завершения (exitCode = 0)
	result := CommandResponse{
		ExitCode:       -2, // -2 означает "команда еще не завершена", реальный exitCode будет доступен после завершения
		StdoutStreamID: stdoutStreamIDInt32,
		StderrStreamID: stderrStreamIDInt32,
	}

	// Сохраняем в хранилище для последующего обновления exitCode
	commandResponses.mu.Lock()
	commandResponses.data[stdoutStreamID] = &result
	commandResponses.mu.Unlock()

	response = result
	return
}

func validateCommand(info plugin.Info, command string) (err error) {

	if len(info.AllowedShellCMDs) == 0 {
		return fmt.Errorf(i18n.Msg("command %s is not allowed: plugin has no allowed commands"), command)
	}

	for _, allowed := range info.AllowedShellCMDs {
		if allowed == command {
			return nil
		}
	}

	return fmt.Errorf(i18n.Msg("command %s is not allowed: not found in allowed commands list"), command)
}

// HostGetStreamReadBufferPtr: streamID — ID потока.
// bufferPtrPtr - указатель на uint32, куда будет записан указатель на буфер
// Возвращает: 0 - успех, иначе код ошибки
func HostGetStreamReadBufferPtr(ctx context.Context, h *host.Host, streamID uint32, bufferPtrPtr uint32) (resultCode uint32) {

	// Проверяем доступность реестра потоков
	if h.StreamRegistry == nil {
		return 1
	}

	// Получаем указатель на буфер из реестра потоков
	bufferPtr, ok := h.StreamRegistry.GetReadBufferPtr(streamID)
	if !ok || bufferPtr == 0 {
		return 1
	}

	// Записываем указатель в память (little-endian uint32)
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

	commandResponses.mu.RLock()
	resp, exists := commandResponses.data[streamID]
	commandResponses.mu.RUnlock()

	if !exists {
		// Если CommandResponse еще не доступен, возвращаем пустой ответ
		resp = &CommandResponse{ExitCode: -1}
	}

	return memory.WriteObjectToPtrSize(ctx, h, resp, resultPtrPtr, resultSizePtr)
}
