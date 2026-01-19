// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package yamlstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	defaultDatabaseVersion = "1.0"
	fileModeDir            = 0755
	fileModeFile           = 0600
)

// YAMLCollectionStore предоставляет thread-safe хранилище для коллекций с кешированием в памяти.
// T - тип элемента коллекции, ID - тип идентификатора элемента.
type YAMLCollectionStore[T any, ID comparable] struct {
	filePath  string
	mu        sync.RWMutex
	cache     []T
	loaded    bool
	getID     func(T) ID
	version   string
	updatedAt time.Time
}

func NewYAMLCollectionStore[T any, ID comparable](filePath string, getID func(T) ID) (store *YAMLCollectionStore[T, ID]) {
	return &YAMLCollectionStore[T, ID]{
		filePath: filePath,
		cache:    make([]T, 0),
		loaded:   false,
		getID:    getID,
		version:  defaultDatabaseVersion,
	}
}

// Load загружает данные из файла (ленивая загрузка при первом обращении).
func (s *YAMLCollectionStore[T, ID]) Load() (err error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded {
		return
	}

	var statErr error
	if _, statErr = os.Stat(s.filePath); os.IsNotExist(statErr) {
		s.cache = make([]T, 0)
		s.version = defaultDatabaseVersion
		s.loaded = true
		return
	}

	var data []byte
	if data, err = os.ReadFile(s.filePath); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to read file: %w"), err)
		return
	}

	if len(data) == 0 {
		s.cache = make([]T, 0)
		s.version = defaultDatabaseVersion
		s.loaded = true
		return
	}

	var db struct {
		Version   string    `yaml:"version"`
		Installed []T       `yaml:"installed"`
		UpdatedAt time.Time `yaml:"updated_at"`
	}

	if err = yaml.Unmarshal(data, &db); err != nil {
		s.cache = make([]T, 0)
		s.version = defaultDatabaseVersion
		s.loaded = true
		return
	}

	s.cache = db.Installed
	if db.Version == "" {
		s.version = defaultDatabaseVersion
	} else {
		s.version = db.Version
	}
	s.updatedAt = db.UpdatedAt
	s.loaded = true

	return
}

// FindByID находит элемент по ID без чтения всего файла.
func (s *YAMLCollectionStore[T, ID]) FindByID(id ID) (item T, found bool, err error) {

	if err = s.Load(); err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, elem := range s.cache {
		if s.getID(elem) == id {
			item = elem
			found = true
			return
		}
	}

	return
}

// Add добавляет или обновляет элемент в кеше.
func (s *YAMLCollectionStore[T, ID]) Add(item T) (err error) {

	if err = s.Load(); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	itemID := s.getID(item)

	for i, elem := range s.cache {
		if s.getID(elem) == itemID {
			s.cache[i] = item
			return
		}
	}

	s.cache = append(s.cache, item)

	return
}

// Remove удаляет элемент из кеша по ID.
func (s *YAMLCollectionStore[T, ID]) Remove(id ID) (err error) {

	if err = s.Load(); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newCache := make([]T, 0, len(s.cache))
	for _, elem := range s.cache {
		if s.getID(elem) != id {
			newCache = append(newCache, elem)
		}
	}

	s.cache = newCache

	return
}

func (s *YAMLCollectionStore[T, ID]) GetAll() (items []T, err error) {

	if err = s.Load(); err != nil {
		items = nil
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]T, len(s.cache))
	copy(result, s.cache)

	items = result
	return
}

// Save сохраняет кеш на диск.
func (s *YAMLCollectionStore[T, ID]) Save() (err error) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err = os.MkdirAll(filepath.Dir(s.filePath), fileModeDir); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "directory", err)
		return
	}

	var db struct {
		Version   string    `yaml:"version"`
		Installed []T       `yaml:"installed"`
		UpdatedAt time.Time `yaml:"updated_at"`
	}

	db.Version = s.version
	db.Installed = make([]T, len(s.cache))
	copy(db.Installed, s.cache)
	db.UpdatedAt = time.Now()

	var data []byte
	if data, err = yaml.Marshal(db); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to marshal: %w"), err)
		return
	}

	if err = os.WriteFile(s.filePath, data, fileModeFile); err != nil {
		err = fmt.Errorf(i18n.Msg("failed to write file: %w"), err)
		return
	}

	return
}

// InvalidateCache инвалидирует кеш, принудительно перезагружая данные при следующем обращении.
func (s *YAMLCollectionStore[T, ID]) InvalidateCache() {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.loaded = false
	s.cache = make([]T, 0)
}
