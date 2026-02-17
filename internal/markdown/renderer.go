// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package markdown

import (
	"github.com/charmbracelet/glamour"
)

func createRenderer(opts *renderOptions) (renderer *glamour.TermRenderer, err error) {

	width := calculateWidth(opts)
	glamourOpts := buildGlamourOptions(opts, width)

	if renderer, err = glamour.NewTermRenderer(glamourOpts...); err != nil {
		return
	}

	return
}

func buildGlamourOptions(opts *renderOptions, width int) (glamourOpts []glamour.TermRendererOption) {

	glamourOpts = append(glamourOpts, glamour.WithWordWrap(width))
	// Убираем WithPreservedNewLines() чтобы переносы строк внутри списков обрабатывались корректно
	// glamourOpts = append(glamourOpts, glamour.WithPreservedNewLines())

	if opts.EnvironmentConfig {
		glamourOpts = append(glamourOpts, glamour.WithEnvironmentConfig())
	}

	if opts.AutoStyle {
		glamourOpts = append(glamourOpts, glamour.WithAutoStyle())
		return
	}

	if opts.Style != "" {
		glamourOpts = append(glamourOpts, glamour.WithStandardStyle(opts.Style))
		return
	}

	return
}
