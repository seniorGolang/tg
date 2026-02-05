// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package log

import (
	"context"
	"log/slog"

	"github.com/goccy/go-json"

	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// parseLogMessage разбирает JSON закодированное сообщение лога на уровень, сообщение и ключ-значение пары.
// Возвращает уровень, сообщение и массив аргументов для slog (чередующиеся ключ-значение).
func parseLogMessage(msgBytes string) (level slog.Level, message string, args []any) {

	// Декодируем JSON сообщение
	var logData logMessage
	if err := json.Unmarshal([]byte(msgBytes), &logData); err != nil {
		// Если не удалось распарсить, пытаемся интерпретировать как простое сообщение
		level = slog.LevelInfo
		message = msgBytes
		return
	}

	// Преобразуем строковый уровень в slog.Level
	switch logData.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Если сообщение пустое, используем исходную строку
	if logData.Message == "" {
		message = msgBytes
		return
	}

	message = logData.Message

	// Преобразуем attrs в массив аргументов (чередующиеся ключ-значение)
	if len(logData.Attrs) == 0 {
		return
	}
	attrs := logData.Attrs

	args = make([]any, 0, len(attrs)*2)
	for k, v := range attrs {
		args = append(args, k, v)
	}

	return
}

// HostLog обрабатывает вызов функции логирования из плагина.
func HostLog(ctx context.Context, logger plugin.Logger, h *host.Host, msgPtr uint32, msgLen uint32) {

	msg, err := memory.ReadString(h, msgPtr, msgLen)
	if err != nil {
		return
	}

	level, message, args := parseLogMessage(msg)

	if h.MuteLogs && level == slog.LevelDebug {
		return
	}

	if len(args) > 0 {
		switch level {
		case slog.LevelDebug:
			logger.Debug(message, args...)
		case slog.LevelInfo:
			logger.Info(message, args...)
		case slog.LevelWarn:
			logger.Warn(message, args...)
		case slog.LevelError:
			logger.Error(message, args...)
		default:
			logger.Info(message, args...)
		}
	} else {
		switch level {
		case slog.LevelDebug:
			logger.Debug(message)
		case slog.LevelInfo:
			logger.Info(message)
		case slog.LevelWarn:
			logger.Warn(message)
		case slog.LevelError:
			logger.Error(message)
		default:
			logger.Info(message)
		}
	}
}
