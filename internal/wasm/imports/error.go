// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package imports

// PluginError представляет ошибку, возвращаемую плагином.
type PluginError struct {
	Message string
}

// Error реализует интерфейс error.
func (e *PluginError) Error() (message string) {

	if e == nil {
		return ""
	}
	return e.Message
}

// NewPluginError создает новую ошибку плагина.
func NewPluginError(message string) (err *PluginError) {
	return &PluginError{Message: message}
}
