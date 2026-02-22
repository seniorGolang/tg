// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package state

import "time"

type PluginState struct {
	Result     PluginExecutionResult
	Options    map[string]any
	PluginName string
	ExecutedAt time.Time
}

type PluginExecutionResult struct {
	Error   string
	Success bool
	Message string
}
