// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package logger

import "sync"

type LogBuffer struct {
	PluginName string
	logs       []string
	mu         sync.Mutex
}

func (b *LogBuffer) GetLogs() (result []string) {

	b.mu.Lock()
	defer b.mu.Unlock()

	result = make([]string, len(b.logs))
	copy(result, b.logs)
	return
}
