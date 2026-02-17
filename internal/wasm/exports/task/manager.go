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

type taskState struct {
	id        uint32
	cancel    context.CancelFunc
	isRunning bool
	module    api.Module
	mu        sync.Mutex
}

func (ts *taskState) GetID() (id uint32) {

	return ts.id
}

func (ts *taskState) GetIsRunning() (isRunning bool) {

	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.isRunning
}

func (ts *taskState) Lock() {

	if ts == nil {
		return
	}
	ts.mu.Lock()
}

func (ts *taskState) Unlock() {

	if ts == nil {
		return
	}
	ts.mu.Unlock()
}

type Manager struct {
	mu          sync.RWMutex
	tasks       map[uint32]*taskState
	nextID      uint32
	callChannel *host.CallChannel
	muteLogs    bool
}

func NewManager(callChannel *host.CallChannel, muteLogs bool) (manager *Manager) {

	return &Manager{
		tasks:       make(map[uint32]*taskState),
		nextID:      1,
		callChannel: callChannel,
		muteLogs:    muteLogs,
	}
}

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

	if !m.muteLogs {
		slog.Debug(i18n.Msg("Task created"),
			slog.Uint64("taskID", uint64(taskID)),
			slog.Uint64("handlerID", uint64(handlerID)),
			slog.String("interval", interval.String()),
		)
	}

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

			if !m.muteLogs {
				slog.Debug(i18n.Msg("Task finished"),
					slog.Uint64("taskID", uint64(taskID)),
					slog.Uint64("handlerID", uint64(handlerID)),
				)
			}
		}()

		for {
			timer := time.NewTimer(interval)
			select {
			case <-taskCtx.Done():
				timer.Stop()
				return
			case <-timer.C:
				if !m.muteLogs {
					slog.Debug(i18n.Msg("Task tick"),
						slog.Uint64("taskID", uint64(taskID)),
						slog.Uint64("handlerID", uint64(handlerID)),
					)
				}
				if m.callHandler(taskCtx, handlerID) {
					return
				}
			}
		}
	}()

	return
}

func (m *Manager) callHandler(ctx context.Context, handlerID uint32) (shouldStop bool) {

	if ctx.Err() != nil {
		return true
	}

	if m.callChannel == nil {
		slog.Error(i18n.Msg("Task callChannel is nil"), slog.Uint64("handlerID", uint64(handlerID)))
		return true
	}

	handlerIDBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(handlerIDBytes, handlerID)
	resultChan := m.callChannel.Call(taskHandlerFunctionName, handlerIDBytes)

	select {
	case <-ctx.Done():
		return true
	case result := <-resultChan:
		if result.Error != nil {
			slog.Error(i18n.Msg("Task handler call failed"), slog.Uint64("handlerID", uint64(handlerID)), slog.Any("error", result.Error))
			return true
		}

		nextValue := uint32(result.Result & 0xFFFFFFFF) //nolint:gosec // Безопасное преобразование через маску
		if nextValue == 0 {
			if !m.muteLogs {
				slog.Debug(i18n.Msg("Task handler returned false, stopping task"), slog.Uint64("handlerID", uint64(handlerID)))
			}
			return true
		}
	}

	return false
}

func (m *Manager) StopTask(taskID uint32) (stopped bool) {

	m.mu.RLock()
	var state *taskState
	var exists bool
	if state, exists = m.tasks[taskID]; !exists {
		m.mu.RUnlock()
		return false
	}
	m.mu.RUnlock()

	state.mu.Lock()
	if !state.isRunning {
		state.mu.Unlock()
		return false
	}
	state.mu.Unlock()

	state.cancel()

	if !m.muteLogs {
		slog.Debug(i18n.Msg("Task stop requested"),
			slog.Uint64("taskID", uint64(taskID)),
		)
	}

	return true
}

func (m *Manager) GetTask(taskID uint32) (state host.TaskStateInterface, exists bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists = m.tasks[taskID]
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
