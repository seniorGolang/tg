// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	reservedLinesCount = 4
	minHeight          = 1
	secondsInMinute    = 60
)

// GetMaxHeightForSelect: по высоте терминала и количеству элементов.
func GetMaxHeightForSelect(itemCount int) (maxHeight int) {

	maxHeight = itemCount
	var height int
	var err error
	if _, height, err = term.GetSize(int(os.Stdout.Fd())); err == nil && height > 0 {
		maxHeight = height - reservedLinesCount
		if maxHeight < minHeight {
			maxHeight = minHeight
		}
		if maxHeight > itemCount {
			maxHeight = itemCount
		}
	}
	return
}

// FormatDuration форматирует длительность в читаемый вид.
func FormatDuration(d time.Duration) (result string) {

	if d < time.Second {
		result = fmt.Sprintf("%dms", d.Milliseconds())
		return
	}
	if d < time.Minute {
		result = fmt.Sprintf("%.1fs", d.Seconds())
		return
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % secondsInMinute
	if seconds == 0 {
		result = fmt.Sprintf("%dm", minutes)
		return
	}
	result = fmt.Sprintf("%dm %ds", minutes, seconds)
	return
}
