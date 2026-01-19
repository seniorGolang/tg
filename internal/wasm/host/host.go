// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package host

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

// netManagerInterface определяет интерфейс для управления сетевыми соединениями.
type netManagerInterface interface {
	GetConn(connID uint64) (net.Conn, error)
	StoreConn(conn net.Conn) uint64
	DelConn(connID uint64)
	GetListener(listenerID uint64) (net.Listener, error)
	StoreListener(listener net.Listener) uint64
	DelListener(listenerID uint64)
}

// Host управляет выполнением WASM плагина.
type Host struct {
	Runtime wazero.Runtime
	Module  api.Module
	Info    plugin.Info
	RootDir string
	Logger  plugin.Logger

	// Функции управления памятью
	Malloc api.Function
	Free   api.Function

	// CompiledModule для переиспользования при создании новых instances
	CompiledModule wazero.CompiledModule

	// NetManager управляет сетевыми соединениями и слушателями
	NetManager netManagerInterface

	// WaitGroup для отслеживания активных HTTP серверов
	// Execute не завершится, пока есть активные серверы
	ActiveServers sync.WaitGroup

	// WaitGroup для отслеживания активных listener'ов
	// Execute не завершится, пока есть активные listener'ы
	ActiveListeners sync.WaitGroup

	// CallChannel - глобальный канал для всех вызовов WASM функций из хоста
	// Обеспечивает последовательное выполнение всех WASM функций
	CallChannel *CallChannel

	// TLSConfig - конфигурация TLS для сетевых соединений
	TLSConfig TLSConfig

	// StreamRegistry управляет потоками данных через кольцевые буферы
	StreamRegistry streamRegistryInterface

	// TaskManager управляет фоновыми задачами
	TaskManager taskManagerInterface

	// WaitGroup для отслеживания активных фоновых задач
	// Execute не завершится, пока есть активные задачи
	ActiveTasks sync.WaitGroup
}

// streamRegistryInterface определяет интерфейс для управления потоками данных.
type streamRegistryInterface interface {
	NewStream(ctx context.Context, h *Host, reader io.Reader, writer io.Writer, bufferSize uint32) (streamID uint32, err error)
	GetStream(streamID uint32) any      // Возвращает *StreamState из пакета stream
	GetStreamState(streamID uint32) any // Возвращает *StreamState из пакета stream
	CloseStream(ctx context.Context, h *Host, streamID uint32)
	GetBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	GetReadBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	GetWriteBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	WaitReaderDone(streamID uint32) // Ждет завершения StartReader() для указанного потока
}

// taskManagerInterface определяет интерфейс для управления фоновыми задачами.
type taskManagerInterface interface {
	StartTask(ctx context.Context, interval time.Duration, handlerID uint32, module api.Module, activeTasks *sync.WaitGroup) (taskID uint32)
	StopTask(taskID uint32) (stopped bool)
	GetTask(taskID uint32) (state TaskStateInterface, exists bool)
	StopAll()
}

// TaskStateInterface определяет интерфейс для состояния задачи.
type TaskStateInterface interface {
	GetID() uint32
	GetIsRunning() bool
	Lock()
	Unlock()
}
