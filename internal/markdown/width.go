// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package markdown

import (
	"os"

	"golang.org/x/term"
)

func calculateWidth(opts *renderOptions) (width int) {

	terminalWidth := getTerminalWidth()

	if opts.WidthPercent > 0 {
		width = (terminalWidth * opts.WidthPercent) / percentBase
		if width < minWidth {
			width = minWidth
		}
		return
	}

	if opts.Width > 0 {
		return opts.Width
	}

	return terminalWidth
}

func getTerminalWidth() (width int) {

	width = defaultTerminalWidth
	var w int
	var err error
	if w, _, err = term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}
	return
}

func CalculateTableWidth(header []string, rows [][]string) (width int) {

	if len(header) == 0 {
		return 0
	}

	columnWidths := make([]int, len(header))
	for i, h := range header {
		columnWidths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(columnWidths) {
				cellLen := len(cell)
				if cellLen > columnWidths[i] {
					columnWidths[i] = cellLen
				}
			}
		}
	}

	totalWidth := 0
	for _, w := range columnWidths {
		totalWidth += w
	}

	separatorsWidth := tableSeparatorStart + (len(columnWidths)-1)*tableSeparatorBetween + tableSeparatorEnd
	width = totalWidth + separatorsWidth + tableReadabilityMargin

	return
}
