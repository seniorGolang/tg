// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package task

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

const (
	responseKeyTaskID        = "taskID"
	responseKeyStatus        = "status"
	responseKeyInterval      = "interval"
	responseStatusStarted    = "started"
	responseStatusStopped    = "stopped"
	responseStatusAllStopped = "all_stopped"
)

// taskResponse представляет JSON формат ответа задачи.
type taskResponse struct {
	Response plugin.Storage `json:"response,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func writeTaskSuccessResponse(ctx context.Context, h *host.Host, data plugin.Storage, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {

	response := taskResponse{
		Response: data,
	}
	var err error
	var responseBytes []byte
	if responseBytes, err = json.Marshal(response); err != nil {
		return 1
	}

	return writeTaskResponse(ctx, h, responseBytes, resultPtrPtr, resultSizePtr)
}

func writeTaskErrorResponse(ctx context.Context, h *host.Host, errorMsg string, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {

	response := taskResponse{
		Error: errorMsg,
	}
	var err error
	var responseBytes []byte
	if responseBytes, err = json.Marshal(response); err != nil {
		return 1
	}

	return writeTaskResponse(ctx, h, responseBytes, resultPtrPtr, resultSizePtr)
}

func writeTaskResponse(ctx context.Context, h *host.Host, responseBytes []byte, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {

	// Проверяем переполнение при преобразовании int -> uint32
	if len(responseBytes) > int(^uint32(0)) {
		return 1
	}
	dataSize := uint32(len(responseBytes)) //nolint:gosec // Проверка переполнения выполнена выше
	if dataSize == 0 {
		return 1
	}

	// Выделяем память
	var err error
	var mallocResults []uint64
	if mallocResults, err = h.Malloc.Call(ctx, uint64(dataSize)); err != nil || len(mallocResults) == 0 {
		return 1
	}

	// Проверяем переполнение при преобразовании uint64 -> uint32
	if mallocResults[0] > uint64(^uint32(0)) {
		return 1
	}
	dataPtr := uint32(mallocResults[0]) //nolint:gosec // Проверка переполнения выполнена выше

	// Гарантируем освобождение памяти при ошибках
	shouldFree := true
	defer func() {
		if shouldFree {
			_, _ = h.Free.Call(ctx, uint64(dataPtr))
		}
	}()

	// Записываем данные
	if err = memory.Write(h, dataPtr, responseBytes); err != nil {
		return 1
	}

	// Записываем указатель и размер
	ptrBytes := make([]byte, 4)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ptrBytes, dataPtr)
	binary.LittleEndian.PutUint32(sizeBytes, dataSize)

	if err = memory.Write(h, resultPtrPtr, ptrBytes); err != nil {
		return 1
	}
	if err = memory.Write(h, resultSizePtr, sizeBytes); err != nil {
		return 1
	}

	// Успешно записали, не освобождаем память (она будет использоваться WASM модулем)
	shouldFree = false
	return 0
}

// HostStartTask запускает фоновую задачу, которая будет выполняться с указанным интервалом.
// Сигнатура: (ctx, m, intervalMs, handlerID, resultPtrPtr, resultSizePtr) -> resultCode
// intervalMs - интервал между выполнениями в миллисекундах
// handlerID - идентификатор функции-обработчика (task_handler будет вызван с этим ID)
// task_handler должен иметь сигнатуру: task_handler(handlerIDPtr uint32, handlerIDSize uint32) -> next uint64
// где next: нижние 32 бита содержат next (1 = true, 0 = false)
// Возвращает taskID для остановки задачи
func HostStartTask(ctx context.Context, h *host.Host, intervalMs, handlerID, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {

	// Читаем входные данные
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval <= 0 {
		return writeTaskErrorResponse(ctx, h, i18n.Msg("interval must be greater than 0"), resultPtrPtr, resultSizePtr)
	}

	h.ActiveTasks.Add(1)

	taskID := h.TaskManager.StartTask(ctx, interval, handlerID, h.Module, &h.ActiveTasks)

	if !h.MuteLogs {
		slog.Debug(i18n.Msg("Task started"),
			slog.Uint64("taskID", uint64(taskID)),
			slog.Uint64("handlerID", uint64(handlerID)),
			slog.String("interval", interval.String()),
		)
	}

	response := plugin.NewStorage()
	_ = response.Set(responseKeyTaskID, taskID)
	_ = response.Set(responseKeyStatus, responseStatusStarted)
	_ = response.Set(responseKeyInterval, interval.String())

	return writeTaskSuccessResponse(ctx, h, response, resultPtrPtr, resultSizePtr)
}

// HostStopTask останавливает задачу по taskID.
// Сигнатура: (ctx, m, taskIDPtr, resultPtrPtr, resultSizePtr) -> resultCode
func HostStopTask(ctx context.Context, h *host.Host, taskIDPtr, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {

	// Читаем taskID из памяти
	var err error
	var taskIDBytes []byte
	if taskIDBytes, err = memory.Read(h, taskIDPtr, 4); err != nil {
		return writeTaskErrorResponse(ctx, h, i18n.Msg("failed to read taskID"), resultPtrPtr, resultSizePtr)
	}
	taskID := binary.LittleEndian.Uint32(taskIDBytes)

	// Останавливаем задачу
	stopped := h.TaskManager.StopTask(taskID)
	if !stopped {
		return writeTaskErrorResponse(ctx, h, fmt.Sprintf(i18n.Msg("task not found or already stopped: %d"), taskID), resultPtrPtr, resultSizePtr)
	}

	if !h.MuteLogs {
		slog.Debug(i18n.Msg("Task stopped"),
			slog.Uint64("taskID", uint64(taskID)),
		)
	}

	response := plugin.NewStorage()
	_ = response.Set(responseKeyTaskID, taskID)
	_ = response.Set(responseKeyStatus, responseStatusStopped)

	return writeTaskSuccessResponse(ctx, h, response, resultPtrPtr, resultSizePtr)
}

// HostStopAll останавливает все активные задачи.
// Сигнатура: (ctx, m, resultPtrPtr, resultSizePtr) -> resultCode
func HostStopAll(ctx context.Context, h *host.Host, resultPtrPtr, resultSizePtr uint32) (resultCode uint32) {

	// Останавливаем все задачи
	h.TaskManager.StopAll()

	if !h.MuteLogs {
		slog.Debug(i18n.Msg("All tasks stopped"))
	}

	response := plugin.NewStorage()
	_ = response.Set(responseKeyStatus, responseStatusAllStopped)

	return writeTaskSuccessResponse(ctx, h, response, resultPtrPtr, resultSizePtr)
}
