// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// Root — канонический skill в InstallPrefix.
type Root struct {
	Name string
	Path string
}

// Activate публикует skills в выбранные targets. skipped — имена targets без существующего корня.
func Activate(home string, roots []Root, opts Options) (states []models.SkillState, skipped []string, err error) {

	states = make([]models.SkillState, 0, len(roots))
	if !opts.Enabled || len(roots) == 0 {
		for _, root := range roots {
			states = append(states, models.SkillState{Name: root.Name, Root: root.Path})
		}
		return
	}

	var targets []Target
	if targets, skipped, err = ResolveTargets(home, opts.Targets, opts.Mkdir); err != nil {
		return nil, nil, err
	}

	for _, root := range roots {
		state := models.SkillState{
			Name:      root.Name,
			Root:      root.Path,
			Published: make([]string, 0, len(targets)),
		}
		for _, target := range targets {
			dest := filepath.Join(target.Skills, root.Name)
			if err = Publish(root.Path, dest); err != nil {
				return nil, skipped, fmt.Errorf("publish skill %s to %s: %w", root.Name, target.Name, err)
			}
			state.Published = append(state.Published, dest)
		}
		states = append(states, state)
	}

	return
}

// Deactivate удаляет ранее опубликованные копии skills.
func Deactivate(states []models.SkillState) (err error) {

	for _, state := range states {
		for _, published := range state.Published {
			if err = Remove(published); err != nil {
				return fmt.Errorf("remove published skill %s: %w", published, err)
			}
		}
	}
	return
}

// Home возвращает домашний каталог пользователя.
func Home() (home string, err error) {

	if home, err = os.UserHomeDir(); err != nil {
		return "", err
	}
	return
}
