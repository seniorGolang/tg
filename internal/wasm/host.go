// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package wasm

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm/env"
	"github.com/seniorGolang/tg/v3/internal/wasm/exports/command"
	"github.com/seniorGolang/tg/v3/internal/wasm/exports/interactive"
	"github.com/seniorGolang/tg/v3/internal/wasm/exports/log"
	"github.com/seniorGolang/tg/v3/internal/wasm/exports/net"
	"github.com/seniorGolang/tg/v3/internal/wasm/exports/task"
	"github.com/seniorGolang/tg/v3/internal/wasm/fs"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/stream"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	// maxMemoryPages — лимит линейной памяти WASM. Страница = 64KB; 65536 страниц = 4GB.
	maxMemoryPages = 65536
)

func New(ctx context.Context, wasmBytes []byte, info plugin.Info, rootDir string, logger plugin.Logger, opts ...Option) (h *host.Host, err error) {

	hostOpts := &hostOptions{}
	for _, opt := range opts {
		opt(hostOpts)
	}

	tlsConfig := host.DefaultTLSConfig()
	if hostOpts.TLSConfig != nil {
		tlsConfig = *hostOpts.TLSConfig
	}

	runtimeConfig := wazero.NewRuntimeConfig().WithMemoryLimitPages(maxMemoryPages)

	if hostOpts.CompilationCache != nil {
		runtimeConfig = runtimeConfig.WithCompilationCache(hostOpts.CompilationCache)
	}

	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	netManager := net.NewNetManager()
	httpManager := net.NewHTTPManager()
	streamRegistry := stream.NewStreamRegistry()

	h = &host.Host{
		Runtime:        r,
		Info:           info,
		RootDir:        rootDir,
		Logger:         logger,
		NetManager:     netManager,
		TLSConfig:      tlsConfig,
		StreamRegistry: streamRegistry,
		MuteLogs:       hostOpts.MuteLogs,
	}

	callChannel := host.NewCallChannel(ctx, h)
	taskManager := task.NewManager(callChannel, hostOpts.MuteLogs)
	h.CallChannel = callChannel
	h.TaskManager = taskManager

	envBuilder := r.NewHostModuleBuilder(ModuleEnv)
	log.RegisterLogFunctions(envBuilder, h.Logger, h)
	interactive.RegisterInteractiveFunctions(envBuilder, h)
	task.RegisterTaskFunctions(envBuilder, h)

	if _, err = envBuilder.Instantiate(ctx); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to instantiate env module: %w"), err)
	}

	netBuilder := r.NewHostModuleBuilder(ModuleNet)
	net.RegisterNetFunctions(netBuilder, h, netManager, httpManager)

	if _, err = netBuilder.Instantiate(ctx); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to instantiate net module: %w"), err)
	}

	commandBuilder := r.NewHostModuleBuilder(ModuleCommand)
	command.RegisterCommandFunctions(commandBuilder, h)

	if _, err = commandBuilder.Instantiate(ctx); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to instantiate command module: %w"), err)
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	var compiledModule wazero.CompiledModule
	if compiledModule, err = r.CompileModule(ctx, wasmBytes); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to compile WASM module: %w"), err)
	}

	h.CompiledModule = compiledModule

	fsBuilder := fs.NewBuilder(rootDir, hostOpts.TGPath)
	fsConfig, resolvedMounts, fsErr := fsBuilder.Build(info)
	if fsErr != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to build filesystem config: %w"), fsErr)
	}

	cfg := wazero.NewModuleConfig().
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithFSConfig(fsConfig).
		WithRandSource(rand.Reader).
		WithStartFunctions(FuncInitialize)

	if info.AllowedStdOut {
		cfg = cfg.WithStdout(os.Stdout)
	}

	if info.AllowedStdErr {
		cfg = cfg.WithStderr(os.Stderr)
	}

	cfg = env.Apply(cfg, h.Info.AllowedEnvVars)
	if !hostOpts.MuteLogs {
		logPluginPermissions(h.Info, resolvedMounts)
	}

	var module api.Module
	if module, err = r.InstantiateModule(ctx, compiledModule, cfg); err != nil {
		return nil, fmt.Errorf(i18n.Msg("failed to instantiate WASM module: %w"), err)
	}

	h.Module = module

	h.Malloc = module.ExportedFunction(FuncMalloc)
	if h.Malloc == nil {
		return nil, errors.New(i18n.Msg("malloc function is not exported from WASM module"))
	}

	h.Free = module.ExportedFunction(FuncFree)
	if h.Free == nil {
		return nil, errors.New(i18n.Msg("free function is not exported from WASM module"))
	}

	netManager.SetOnNewConnectionCallback(func(connID uint64) {
		_ = h.CallChannel.CallWithUint64("on_new_connection", connID)
	})

	return
}

func Close(ctx context.Context, h *host.Host) (err error) {

	if h.CallChannel != nil {
		h.CallChannel.Close()
	}

	if h.Runtime != nil {
		return h.Runtime.Close(ctx)
	}
	return
}

func logPluginPermissions(info plugin.Info, resolved []fs.ResolvedMount) {

	if len(resolved) > 0 {
		pathsList := make([]string, 0, len(resolved))
		for _, r := range resolved {
			pathsList = append(pathsList, fmt.Sprintf("%s (%s) -> %s", r.PathKey, r.AccessLevel, r.MountPoint))
		}
		slog.Debug(i18n.Msg("Plugin filesystem permissions"), "plugin", info.Name, "paths", pathsList)
	} else {
		slog.Debug(i18n.Msg("Plugin filesystem permissions"), "plugin", info.Name, "paths", []string{})
	}

	if len(info.AllowedEnvVars) > 0 {
		envVars := env.ResolveEnvVars(info.AllowedEnvVars)
		envKeys := make([]string, 0, len(envVars))
		for key := range envVars {
			envKeys = append(envKeys, key)
		}
		slog.Debug(i18n.Msg("Plugin environment variables"), "plugin", info.Name, "variables", envKeys)
	} else {
		slog.Debug(i18n.Msg("Plugin environment variables"), "plugin", info.Name, "variables", []string{})
	}

	if len(info.Commands) > 0 {
		commandPaths := make([]string, 0, len(info.Commands))
		for _, cmd := range info.Commands {
			commandPaths = append(commandPaths, strings.Join(cmd.Path, " "))
		}
		slog.Debug(i18n.Msg("Plugin commands"), "plugin", info.Name, "commands", commandPaths)
	} else {
		slog.Debug(i18n.Msg("Plugin commands"), "plugin", info.Name, "commands", []string{})
	}
}
