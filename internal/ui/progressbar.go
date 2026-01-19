// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type ProgressBar struct {
	title      string
	total      int
	current    int
	progress   progress.Model
	termWidth  int
	startTime  time.Time
	titleWidth int
}

var (
	cachedTermWidth int
	termWidthOnce   sync.Once
)

func getCachedTerminalWidth() (width int) {

	termWidthOnce.Do(func() {
		cachedTermWidth = defaultTermWidth
		var w int
		var err error
		if w, _, err = term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			cachedTermWidth = w
		}
	})
	width = cachedTermWidth
	return
}

func NewProgressBar(title string, total int, titleWidth int) (bar *ProgressBar) {

	termWidth := getCachedTerminalWidth()

	prog := progress.New(
		progress.WithWidth(fixedBarWidth),
		progress.WithScaledGradient(gradientStartColor, gradientEndColor),
		progress.WithoutPercentage(),
	)

	bar = &ProgressBar{
		title:      title,
		total:      total,
		current:    0,
		progress:   prog,
		termWidth:  termWidth,
		startTime:  time.Now(),
		titleWidth: titleWidth,
	}
	return
}

func (pb *ProgressBar) UpdateTitle(title string) {

	pb.title = title
}

func (pb *ProgressBar) Increment() {

	if pb.current < pb.total {
		pb.current++
	}
}

func (pb *ProgressBar) SetCurrent(current int) {

	if current < 0 {
		current = 0
	}
	if current > pb.total {
		current = pb.total
	}
	pb.current = current
}

func (pb *ProgressBar) View() (line string) {

	percent := float64(pb.current) / float64(pb.total)
	if percent > maxPercent {
		percent = maxPercent
	}

	elapsed := time.Since(pb.startTime)
	elapsedStr := formatDurationFixed(elapsed)
	rightPart := fmt.Sprintf(rightPartFormat, verticalLine, elapsedStr)

	titleStyle := lipgloss.NewStyle().Width(pb.titleWidth).Align(lipgloss.Left)
	titleFormatted := titleStyle.Render(pb.title)

	var middlePart string
	var rightSectionWidth int

	if percent >= maxPercent {
		checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(checkmarkColor)).Bold(true)
		checkmark := checkmarkStyle.Render(checkmark)
		middlePart = checkmark
		rightSectionWidth = lipgloss.Width(checkmark) + 1 + lipgloss.Width(rightPart)
	} else {
		bar := pb.progress.ViewAs(percent)
		percentStr := fmt.Sprintf(percentFormat, int(percent*100))

		percentWidth := lipgloss.Width(percentStr)
		barWidth := lipgloss.Width(bar)
		rightPartWidth := lipgloss.Width(rightPart)
		rightSectionWidth = percentWidth + 1 + barWidth + 1 + rightPartWidth

		middlePart = lipgloss.JoinHorizontal(
			lipgloss.Left,
			percentStr,
			space,
			bar,
		)
	}

	leftPartWidth := lipgloss.Width(titleFormatted)
	spacesNeeded := pb.termWidth - leftPartWidth - rightSectionWidth
	if spacesNeeded < minSpacesNeeded {
		spacesNeeded = minSpacesNeeded
	}

	line = lipgloss.JoinHorizontal(
		lipgloss.Left,
		titleFormatted,
		strings.Repeat(space, spacesNeeded),
		middlePart,
		space,
		rightPart,
	)

	return
}

func (pb *ProgressBar) Stop() (result string) {

	pb.current = pb.total
	result = pb.View()
	return
}

func (pb *ProgressBar) Print() {

	line := carriageRet + pb.View()
	os.Stdout.WriteString(line)
}

func (pb *ProgressBar) Println() {

	fmt.Println(carriageRet + pb.View())
}
