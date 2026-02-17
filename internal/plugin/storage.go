// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package plugin

import (
	"errors"
	"fmt"

	"github.com/goccy/go-json"
)

const (
	errStorageNil           = "storage is nil"
	errMarshalValueForKey   = "failed to marshal value for key %q: %w"
	errKeyNotFound          = "key %q: %w"
	errUnmarshalValueForKey = "failed to unmarshal value for key %q: %w"
	errKeyNotFoundMessage   = "key not found"
)

var (
	ErrNotFound = errors.New(errKeyNotFoundMessage)
)

type Storage interface {
	Set(name string, value any) (err error)

	Has(name string) (has bool)

	GetRaw(name string) (value json.RawMessage, ok bool)
}

type MapStorage map[string]json.RawMessage

func NewStorage() (storage Storage) {

	s := make(MapStorage)
	return &s
}

func (s MapStorage) GetRaw(name string) (value json.RawMessage, ok bool) {

	if s == nil {
		return nil, false
	}
	value, ok = s[name]
	return
}

func (s MapStorage) Set(name string, value any) (err error) {

	if s == nil {
		return errors.New(errStorageNil)
	}
	var data []byte
	if data, err = json.Marshal(value); err != nil {
		return fmt.Errorf(errMarshalValueForKey, name, err)
	}
	s[name] = data
	return
}

func (s MapStorage) Has(name string) (has bool) {

	if s == nil {
		return false
	}
	_, has = s[name]
	return
}

func Get[T any](store Storage, key string) (value T, err error) {

	if store == nil {
		return value, ErrNotFound
	}
	raw, ok := store.GetRaw(key)
	if !ok {
		return value, fmt.Errorf(errKeyNotFound, key, ErrNotFound)
	}
	if err = json.Unmarshal(raw, &value); err != nil {
		return value, fmt.Errorf(errUnmarshalValueForKey, key, err)
	}
	return
}
