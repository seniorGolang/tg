// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package imports

import (
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

// executeRequest представляет JSON формат запроса для выполнения плагина.
type executeRequest struct {
	Request plugin.Storage `json:"request"`
}

// executeResponse представляет JSON формат ответа от выполнения плагина.
type executeResponse struct {
	Error    string            `json:"error,omitempty"`
	Response plugin.MapStorage `json:"response,omitempty"`
}

// infoRequest представляет JSON формат запроса для получения информации о плагине.
type infoRequest struct{}

// infoResponse представляет JSON формат ответа с информацией о плагине.
type infoResponse struct {
	Info plugin.Info `json:"info"`
}

// generateRequest представляет JSON формат запроса для генерации.
type generateRequest struct {
	RootDir    string `json:"rootDir"`
	ModuleName string `json:"moduleName"`
}

// generateResponse представляет JSON формат ответа от генерации.
type generateResponse struct {
	Error string `json:"error,omitempty"`
}
