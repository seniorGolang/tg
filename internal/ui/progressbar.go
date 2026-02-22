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
	title         string
	total         int
	current       int
	progress      progress.Model
	termWidth     int
	startTime     time.Time
	titleWidth    int
	indeterminate bool
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
		//nolint:gosec // G115: fd stdout на всех платформах в пределах int
		if w, _, err = term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			cachedTermWidth = w
		}
	})
	return cachedTermWidth
}

func NewProgressBar(title string, total int, titleWidth int) (bar *ProgressBar) {

	termWidth := getCachedTerminalWidth()

	prog := progress.New(
		progress.WithWidth(fixedBarWidth),
		progress.WithScaledGradient(gradientStartColor, gradientEndColor),
		progress.WithoutPercentage(),
	)

	return &ProgressBar{
		title:      title,
		total:      total,
		current:    0,
		progress:   prog,
		termWidth:  termWidth,
		startTime:  time.Now(),
		titleWidth: titleWidth,
	}
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

func (pb *ProgressBar) SetIndeterminate(indeterminate bool) {

	pb.indeterminate = indeterminate
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

	switch {
	case pb.indeterminate:
		runes := []rune(spinnerFrames)
		frame := (time.Now().UnixNano() / spinnerIntervalNs) % int64(len(runes))
		spinnerChar := lipgloss.NewStyle().Foreground(lipgloss.Color(spinnerColor)).Bold(true).Render(string(runes[frame]))
		middlePart = spinnerChar
		rightSectionWidth = lipgloss.Width(spinnerChar) + 1 + lipgloss.Width(rightPart)
	case percent >= maxPercent:
		checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(checkmarkColor)).Bold(true)
		mark := checkmarkStyle.Render(checkmark)
		middlePart = mark
		rightSectionWidth = lipgloss.Width(mark) + 1 + lipgloss.Width(rightPart)
	default:
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

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		titleFormatted,
		strings.Repeat(space, spacesNeeded),
		middlePart,
		space,
		rightPart,
	)
}

func (pb *ProgressBar) Stop() (result string) {

	pb.indeterminate = false
	pb.current = pb.total
	return pb.View()
}

func (pb *ProgressBar) Print() {

	line := carriageRet + pb.View()
	os.Stdout.WriteString(line)
}

func (pb *ProgressBar) Println() {

	fmt.Println(carriageRet + pb.View())
}
