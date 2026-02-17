// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package markdown

type renderOptions struct {
	WidthPercent      int
	Width             int
	Style             string
	AutoStyle         bool
	EnvironmentConfig bool
}

type Option func(opts *renderOptions)

func WithWidthPercent(percent int) (opt Option) {

	return func(opts *renderOptions) {
		opts.WidthPercent = percent
	}
}

func WithWidth(width int) (opt Option) {

	return func(opts *renderOptions) {
		opts.Width = width
	}
}

func WithStyle(style string) (opt Option) {

	return func(opts *renderOptions) {
		opts.Style = style
		opts.AutoStyle = false
	}
}

func WithAutoStyle() (opt Option) {

	return func(opts *renderOptions) {
		opts.AutoStyle = true
		opts.Style = ""
	}
}

func WithEnvironmentConfig() (opt Option) {

	return func(opts *renderOptions) {
		opts.EnvironmentConfig = true
	}
}
