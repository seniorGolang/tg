// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"context"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
)

const (
	TargetAgents = "agents"
	TargetCursor = "cursor"
	TargetClaude = "claude"
	TargetCodex  = "codex"
)

// Options управляет активацией skills при установке и reinstall.
type Options struct {
	Enabled bool
	Mkdir   bool
	Targets []string
}

// Default возвращает политику по умолчанию: только agents, без создания каталогов.
func Default() (opts Options) {

	return Options{
		Enabled: true,
		Mkdir:   false,
		Targets: []string{TargetAgents},
	}
}

// FromContext читает Options из ctx; при отсутствии ключа — Default.
func FromContext(ctx context.Context) (opts Options) {

	opts = Default()
	value := ctx.Value(contextkeys.Skills)
	if value == nil {
		return
	}
	parsed, ok := value.(Options)
	if !ok {
		return
	}
	opts = parsed
	if len(opts.Targets) == 0 {
		opts.Targets = []string{TargetAgents}
	}
	return
}

// WithContext кладёт Options в ctx.
func WithContext(ctx context.Context, opts Options) (out context.Context) {

	return context.WithValue(ctx, contextkeys.Skills, opts)
}

// ParseTargets разбирает список targets через запятую.
func ParseTargets(raw string) (targets []string) {

	if raw == "" {
		return []string{TargetAgents}
	}
	parts := strings.Split(raw, ",")
	targets = make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		targets = append(targets, name)
	}
	if len(targets) == 0 {
		return []string{TargetAgents}
	}
	return
}
