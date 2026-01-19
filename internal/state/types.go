// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package state

import "time"

type PluginState struct {
	PluginName string
	Options    map[string]any
	ExecutedAt time.Time
	Result     PluginExecutionResult
}

type PluginExecutionResult struct {
	Success bool
	Error   string
	Message string
}
