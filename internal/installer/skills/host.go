// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	skillfs "github.com/seniorGolang/tg/v3/skills"
)

const (
	hostSkillsDirName = "host"
	hostStateFileName = "host.yml"
)

type hostState struct {
	Skills []models.SkillState `yaml:"skills,omitempty"`
}

// InstallHost извлекает встроенные skills в TG_HOME и публикует их в targets.
func InstallHost(opts Options) (err error) {

	opts.Enabled = true

	canonRoot := filepath.Join(storage.GetHomeDir(), "skills", hostSkillsDirName)
	if err = os.RemoveAll(canonRoot); err != nil {
		return err
	}
	if err = extractHostSkills(canonRoot); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to extract host skills: %w"), err)
	}

	statePath := filepath.Join(storage.GetHomeDir(), "skills", hostStateFileName)
	var previous hostState
	if previous, err = loadHostState(statePath); err != nil {
		return err
	}
	if err = Deactivate(previous.Skills); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to deactivate skills: %w"), err)
	}

	var roots []Root
	if roots, err = scanHostRoots(canonRoot); err != nil {
		return err
	}
	if len(roots) == 0 {
		return
	}

	var home string
	if home, err = Home(); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to resolve home directory: %w"), err)
	}

	var skipped []string
	var states []models.SkillState
	if states, skipped, err = Activate(home, roots, opts); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to activate skills: %w"), err)
	}
	for _, name := range skipped {
		slog.Warn(i18n.Msg("Skills target skipped: root directory not found"), slog.String("target", name))
	}

	return saveHostState(statePath, hostState{Skills: states})
}

func extractHostSkills(destRoot string) (err error) {

	return fs.WalkDir(skillfs.FS, ".", func(path string, entry fs.DirEntry, walkErr error) (err error) {

		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}
		target := filepath.Join(destRoot, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		var data []byte
		if data, err = skillfs.FS.ReadFile(path); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func scanHostRoots(canonRoot string) (roots []Root, err error) {

	var entries []os.DirEntry
	if entries, err = os.ReadDir(canonRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	roots = make([]Root, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(canonRoot, entry.Name())
		if _, statErr := os.Stat(filepath.Join(skillPath, skillFileName)); statErr != nil {
			continue
		}
		roots = append(roots, Root{Name: entry.Name(), Path: skillPath})
	}
	return
}

func loadHostState(path string) (state hostState, err error) {

	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		if os.IsNotExist(err) {
			return hostState{}, nil
		}
		return hostState{}, err
	}
	if err = yaml.Unmarshal(data, &state); err != nil {
		return hostState{}, err
	}
	return
}

func saveHostState(path string, state hostState) (err error) {

	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	var data []byte
	if data, err = yaml.Marshal(&state); err != nil {
		return
	}
	return os.WriteFile(path, data, 0600)
}
