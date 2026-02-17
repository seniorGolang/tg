// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package logger

import (
	"context"
	"log/slog"
	"sync"

	"github.com/pterm/pterm"
)

func init() {

	pterm.DefaultLogger = *pterm.DefaultLogger.WithMaxWidth(pterm.GetTerminalWidth())
}

type PTermHandler struct {
	level slog.Level
	mu    sync.Mutex
}

func NewPTermHandler(level slog.Level) (handler *PTermHandler) {
	return &PTermHandler{level: level}
}

func (h *PTermHandler) Enabled(_ context.Context, level slog.Level) (enabled bool) {
	return level >= h.level
}

func (h *PTermHandler) Handle(_ context.Context, record slog.Record) (err error) {

	var args []any
	record.Attrs(func(a slog.Attr) bool {
		args = append(args, a.Key, a.Value.Any())
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	logger := pterm.DefaultLogger.WithLevel(pterm.LogLevelDebug)

	var loggerArgs []pterm.LoggerArgument
	if len(args) > 0 {
		loggerArgs = make([]pterm.LoggerArgument, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				var key string
				var ok bool
				key, ok = args[i].(string)
				if !ok {
					continue
				}
				loggerArgs = append(loggerArgs, pterm.LoggerArgument{
					Key:   key,
					Value: args[i+1],
				})
			}
		}
	}

	logWithLevel(logger, record.Level, record.Message, loggerArgs)

	return
}

func logWithLevel(logger *pterm.Logger, level slog.Level, msg string, args []pterm.LoggerArgument) {

	if len(args) > 0 {
		switch level {
		case slog.LevelError:
			logger.Error(msg, args)
		case slog.LevelWarn:
			logger.Warn(msg, args)
		case slog.LevelInfo:
			logger.Info(msg, args)
		case slog.LevelDebug:
			logger.Debug(msg, args)
		default:
			logger.Print(msg, args)
		}
	} else {
		switch level {
		case slog.LevelError:
			logger.Error(msg)
		case slog.LevelWarn:
			logger.Warn(msg)
		case slog.LevelInfo:
			logger.Info(msg)
		case slog.LevelDebug:
			logger.Debug(msg)
		default:
			logger.Print(msg)
		}
	}
}

func (h *PTermHandler) WithAttrs(_ []slog.Attr) (handler slog.Handler) {
	return h
}

func (h *PTermHandler) WithGroup(_ string) (handler slog.Handler) {
	return h
}

var _ slog.Handler = (*PTermHandler)(nil)
