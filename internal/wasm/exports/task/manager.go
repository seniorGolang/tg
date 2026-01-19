// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package task

import (
	"context"
	"encoding/binary"
	"log/slog"
	"sync"
	"time"

	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

const (
	taskHandlerFunctionName = "task_handler"
)

// taskState представляет состояние задачи.
type taskState struct {
	id        uint32
	cancel    context.CancelFunc
	isRunning bool
	module    api.Module
	mu        sync.Mutex
}

func (ts *taskState) GetID() (id uint32) {

	id = ts.id
	return
}

func (ts *taskState) GetIsRunning() (isRunning bool) {

	ts.mu.Lock()
	defer ts.mu.Unlock()
	isRunning = ts.isRunning
	return
}

// Lock блокирует мьютекс задачи.
func (ts *taskState) Lock() {

	ts.mu.Lock()
}

// Unlock разблокирует мьютекс задачи.
func (ts *taskState) Unlock() {

	ts.mu.Unlock()
}

// Manager управляет фоновыми задачами.
type Manager struct {
	mu          sync.RWMutex
	tasks       map[uint32]*taskState
	nextID      uint32
	callChannel *host.CallChannel
}

func NewManager(callChannel *host.CallChannel) (manager *Manager) {

	return &Manager{
		tasks:       make(map[uint32]*taskState),
		nextID:      1,
		callChannel: callChannel,
	}
}

// StartTask запускает новую задачу и возвращает её ID.
// handlerID - идентификатор функции-обработчика, которая будет вызываться с интервалом
func (m *Manager) StartTask(ctx context.Context, interval time.Duration, handlerID uint32, module api.Module, activeTasks *sync.WaitGroup) (taskID uint32) {

	m.mu.Lock()
	taskID = m.nextID
	m.nextID++
	taskCtx, cancel := context.WithCancel(ctx)
	state := &taskState{
		id:        taskID,
		cancel:    cancel,
		isRunning: true,
		module:    module,
	}
	m.tasks[taskID] = state
	m.mu.Unlock()

	slog.Debug(i18n.Msg("Task created"),
		slog.Uint64("taskID", uint64(taskID)),
		slog.Uint64("handlerID", uint64(handlerID)),
		slog.String("interval", interval.String()),
	)

	go func() {
		defer func() {
			m.mu.Lock()
			state.mu.Lock()
			state.isRunning = false
			state.mu.Unlock()
			delete(m.tasks, taskID)
			m.mu.Unlock()

			if activeTasks != nil {
				activeTasks.Done()
			}

			slog.Debug(i18n.Msg("Task finished"),
				slog.Uint64("taskID", uint64(taskID)),
				slog.Uint64("handlerID", uint64(handlerID)),
			)
		}()

		for {
			timer := time.NewTimer(interval)
			select {
			case <-taskCtx.Done():
				timer.Stop()
				return
			case <-timer.C:
				slog.Debug(i18n.Msg("Task tick"),
					slog.Uint64("taskID", uint64(taskID)),
					slog.Uint64("handlerID", uint64(handlerID)),
				)
				if m.callHandler(taskCtx, handlerID) {
					return
				}
			}
		}
	}()

	return
}

// callHandler вызывает функцию-обработчик задачи через CallChannel
// Возвращает true, если задача должна быть остановлена (next = false или ошибка)
func (m *Manager) callHandler(ctx context.Context, handlerID uint32) (shouldStop bool) {

	// Проверяем контекст перед вызовом
	if ctx.Err() != nil {
		// Контекст отменен - останавливаем задачу
		return true
	}

	// Проверяем, что callChannel валиден
	if m.callChannel == nil {
		slog.Error(i18n.Msg("Task callChannel is nil"), slog.Uint64("handlerID", uint64(handlerID)))
		return true
	}

	// Сериализуем handlerID в 4 байта (binary.LittleEndian)
	handlerIDBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(handlerIDBytes, handlerID)

	// Вызываем task_handler через CallChannel
	// Сигнатура: task_handler(handlerIDPtr uint32, handlerIDSize uint32) -> next uint64
	// где next: верхние 32 бита не используются, нижние 32 бита содержат next (1 = true, 0 = false)
	resultChan := m.callChannel.Call(taskHandlerFunctionName, handlerIDBytes)

	// Ждем результат с таймаутом через контекст
	select {
	case <-ctx.Done():
		// Контекст отменен - останавливаем задачу
		return true
	case result := <-resultChan:
		if result.Error != nil {
			// Ошибка вызова - останавливаем задачу
			slog.Error(i18n.Msg("Task handler call failed"), slog.Uint64("handlerID", uint64(handlerID)), slog.Any("error", result.Error))
			return true
		}

		// Извлекаем next из результата (нижние 32 бита)
		// Маска 0xFFFFFFFF гарантирует, что значение будет в пределах uint32
		nextValue := uint32(result.Result & 0xFFFFFFFF) //nolint:gosec // Безопасное преобразование через маску
		// nextValue: 1 (true) = продолжаем, 0 (false) = завершаем задачу
		if nextValue == 0 {
			// Обработчик вернул false - завершаем задачу
			slog.Debug(i18n.Msg("Task handler returned false, stopping task"), slog.Uint64("handlerID", uint64(handlerID)))
			return true
		}
		// nextValue != 0 (true) - продолжаем выполнение
	}

	return false
}

// StopTask останавливает задачу по ID.
func (m *Manager) StopTask(taskID uint32) (stopped bool) {

	m.mu.RLock()
	state, exists := m.tasks[taskID]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	state.mu.Lock()
	if !state.isRunning {
		state.mu.Unlock()
		return false
	}
	state.mu.Unlock()

	// Отменяем контекст задачи
	state.cancel()

	slog.Debug(i18n.Msg("Task stop requested"),
		slog.Uint64("taskID", uint64(taskID)),
	)

	// Удаляем задачу из map (будет удалена в defer горутины)
	return true
}

func (m *Manager) GetTask(taskID uint32) (state host.TaskStateInterface, exists bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	taskState, ok := m.tasks[taskID]
	if !ok {
		return
	}
	state = taskState
	exists = true
	return
}

func (m *Manager) GetAllTasks() (tasks []*taskState) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks = make([]*taskState, 0, len(m.tasks))
	for _, state := range m.tasks {
		tasks = append(tasks, state)
	}
	return
}

// StopAll останавливает все активные задачи.
func (m *Manager) StopAll() {

	m.mu.RLock()
	tasks := make([]*taskState, 0, len(m.tasks))
	for _, state := range m.tasks {
		tasks = append(tasks, state)
	}
	m.mu.RUnlock()

	for _, state := range tasks {
		state.cancel()
	}
}
