// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package logger

import (
	"log/slog"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

type SlogAdapter struct {
	logger *slog.Logger
}

func NewSlogAdapter(logger *slog.Logger) (adapter *SlogAdapter) {
	return &SlogAdapter{logger: logger}
}

func (a *SlogAdapter) Debug(msg string, args ...any) {

	if a.logger == nil {
		return
	}
	a.logger.Debug(msg, args...)
}

func (a *SlogAdapter) Info(msg string, args ...any) {

	if a.logger == nil {
		return
	}
	a.logger.Info(msg, args...)
}

func (a *SlogAdapter) Warn(msg string, args ...any) {

	if a.logger == nil {
		return
	}
	a.logger.Warn(msg, args...)
}

func (a *SlogAdapter) Error(msg string, args ...any) {

	if a.logger == nil {
		return
	}
	a.logger.Error(msg, args...)
}

var _ plugin.Logger = (*SlogAdapter)(nil)
