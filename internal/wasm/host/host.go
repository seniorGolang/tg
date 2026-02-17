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
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
	"github.com/seniorGolang/tg/v3/internal/wasm/stream"
)

type Host struct {
	Info    plugin.Info
	Logger  plugin.Logger
	Module  api.Module
	RootDir string
	Runtime wazero.Runtime

	Free   api.Function
	Malloc api.Function

	CompiledModule wazero.CompiledModule

	NetManager netManagerInterface
	TLSConfig  TLSConfig

	CallChannel    *CallChannel
	StreamRegistry streamRegistry
	TaskManager    taskManagerInterface

	ActiveListeners sync.WaitGroup
	ActiveServers   sync.WaitGroup
	ActiveTasks     sync.WaitGroup

	MuteLogs bool
}

type streamRegistry interface {
	NewStream(ctx context.Context, h memory.Host, reader io.Reader, writer io.Writer, bufferSize uint32) (streamID uint32, err error)
	GetStreamState(streamID uint32) (state *stream.StreamState, ok bool)
	CloseStream(ctx context.Context, h memory.Host, streamID uint32)
	GetBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	GetReadBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	GetWriteBufferPtr(streamID uint32) (bufferPtr uint32, ok bool)
	WaitReaderDone(streamID uint32)
}

type netManagerInterface interface {
	DelConn(connID uint64)
	DelListener(listenerID uint64)
	GetConn(connID uint64) (conn net.Conn, err error)
	StoreListener(listener net.Listener) (listenerID uint64)
	GetListener(listenerID uint64) (listener net.Listener, err error)
	StoreConnWithStream(ctx context.Context, h any, conn net.Conn) (connID uint64)
}

type taskManagerInterface interface {
	StartTask(ctx context.Context, interval time.Duration, handlerID uint32, module api.Module, activeTasks *sync.WaitGroup) (taskID uint32)
	StopTask(taskID uint32) (stopped bool)
	GetTask(taskID uint32) (state TaskStateInterface, exists bool)
	StopAll()
}

type TaskStateInterface interface {
	Lock()
	Unlock()
	GetID() (id uint32)
	GetIsRunning() (isRunning bool)
}
