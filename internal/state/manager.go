// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type Manager struct {
	RootDir   string
	stateFile string
	mu        sync.RWMutex
	cache     map[string]PluginState
}

func New(rootDir string) (sm *Manager) {

	sm = &Manager{
		RootDir:   rootDir,
		stateFile: filepath.Join(rootDir, tgDirName, stateFileName),
		cache:     make(map[string]PluginState),
	}

	states, err := sm.loadAllStatesUnsafe()
	if err == nil {
		sm.cache = states
	}

	return
}

func (sm *Manager) SetPluginState(pluginName string, options map[string]any, result PluginExecutionResult) (err error) {

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.cache[pluginName] = PluginState{
		PluginName: pluginName,
		Options:    options,
		ExecutedAt: time.Now(),
		Result:     result,
	}

	return
}

func (sm *Manager) LoadPluginState(pluginName string) (state PluginState, exists bool) {

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists = sm.cache[pluginName]
	return
}

func (sm *Manager) LoadAllStates() (states map[string]PluginState, err error) {

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	states = make(map[string]PluginState, len(sm.cache))
	for k, v := range sm.cache {
		states[k] = v
	}
	return
}

func (sm *Manager) SaveAllStates() (err error) {

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.cache) == 0 {
		return
	}

	states := make(map[string]PluginState, len(sm.cache))
	for k, v := range sm.cache {
		states[k] = v
	}

	return sm.saveAllStatesUnsafe(states)
}

func (sm *Manager) loadAllStatesUnsafe() (states map[string]PluginState, err error) {

	states = make(map[string]PluginState)

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(sm.stateFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to read state file"), err)
		return
	}
	_ = fileInfo

	var data []byte
	data, err = os.ReadFile(sm.stateFile)
	if err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to read state file"), err)
		return
	}

	if len(data) == 0 {
		return
	}

	//nolint:musttag // yaml теги удалены по запросу, используется прямое именование полей
	if err = yaml.Unmarshal(data, &states); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to unmarshal state"), err)
		return
	}

	return
}

func (sm *Manager) saveAllStatesUnsafe(states map[string]PluginState) (err error) {

	stateDir := filepath.Dir(sm.stateFile)
	if err = os.MkdirAll(stateDir, stateDirPerm); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "state directory", err)
		return
	}

	var data []byte
	//nolint:musttag // yaml теги удалены по запросу, используется прямое именование полей
	data, err = yaml.Marshal(states)
	if err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to marshal state"), err)
		return
	}

	if err = os.WriteFile(sm.stateFile, data, stateFilePerm); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to write state file"), err)
		return
	}

	return
}

func (sm *Manager) RemovePluginState(pluginName string) (err error) {

	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.cache, pluginName)
	return
}

func (sm *Manager) ClearAllStates() (err error) {

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.cache = make(map[string]PluginState)
	return
}
