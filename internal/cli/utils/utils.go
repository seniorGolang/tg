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
	minHeight          = 1
	secondsInMinute    = 60
	reservedLinesCount = 4
)

func GetMaxHeightForSelect(itemCount int) (maxHeight int) {

	maxHeight = itemCount
	var err error
	var height int
	//nolint:gosec // G115: fd stdout на всех платформах в пределах int
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

func FormatDuration(d time.Duration) (result string) {

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % secondsInMinute
	if seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
