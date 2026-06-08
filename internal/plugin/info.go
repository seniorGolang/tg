// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package plugin

type Info struct {
	Name        string    `json:"name"`
	Persistent  bool      `json:"persistent"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	License     string    `json:"license"`
	Category    string    `json:"category"`
	Commands    []Command `json:"commands"`
	Group       string    `json:"group,omitempty"`
	Doc         string    `json:"doc"`
	Options     []Option  `json:"options"`

	Kind             string            `json:"kind,omitempty"`
	Silent           bool              `json:"silent,omitempty"`
	AllowedStdOut    bool              `json:"allowed_std_out,omitempty"`
	AllowedStdErr    bool              `json:"allowed_std_err,omitempty"`
	AllowedHosts     []string          `json:"allowedHosts,omitempty"`
	AllowedListeners []string          `json:"allowedListeners,omitempty"`
	AllowedShellCMDs []string          `json:"allowedShellCMDs,omitempty"`
	AllowedEnvVars   []string          `json:"allowedEnvVars,omitempty"`
	Dependencies     []string          `json:"dependencies,omitempty"`
	Always           bool              `json:"always,omitempty"`
	InitPkgs         []string          `json:"initPkgs,omitempty"`
	AllowedPaths     map[string]string `json:"allowedPaths,omitempty"`
}

type Command struct {
	Path        []string `json:"path"`
	Description string   `json:"description"`
	Options     []Option `json:"options"`
}

type Option struct {
	Name         string `json:"name"`
	Short        string `json:"short,omitempty"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	Default      any    `json:"default,omitempty"`
	IsPositional bool   `json:"isPositional,omitempty"`
}
