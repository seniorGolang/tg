// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package markdown

import (
	"github.com/charmbracelet/glamour"
)

func RenderContent(content string, opts ...Option) (rendered string, err error) {

	options := defaultRenderOptions()
	for _, opt := range opts {
		opt(options)
	}

	var renderer *glamour.TermRenderer
	if renderer, err = createRenderer(options); err != nil {
		return
	}

	if rendered, err = renderer.Render(content); err != nil {
		return
	}

	return
}

func defaultRenderOptions() (opts *renderOptions) {

	return &renderOptions{
		EnvironmentConfig: true,
	}
}
