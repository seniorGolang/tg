// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package log

// logMessage представляет структуру JSON сообщения лога.
type logMessage struct {
	Level   string         `json:"level"` // Уровень логирования как строка: "debug", "info", "warn", "error"
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}
