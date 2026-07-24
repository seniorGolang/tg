// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

const skillFileName = "SKILL.md"

// FromSpecs строит Roots из спецификаций манифеста относительно installPrefix.
func FromSpecs(installPrefix string, specs []models.SkillSpec) (roots []Root) {

	roots = make([]Root, 0, len(specs))
	for _, spec := range specs {
		if spec.Name == "" || spec.Root == "" {
			continue
		}
		roots = append(roots, Root{
			Name: spec.Name,
			Path: filepath.Join(installPrefix, spec.Root),
		})
	}
	return
}

// FromStates восстанавливает Roots из сохранённого состояния installation.
func FromStates(states []models.SkillState) (roots []Root) {

	roots = make([]Root, 0, len(states))
	for _, state := range states {
		if state.Name == "" || state.Root == "" {
			continue
		}
		roots = append(roots, Root{Name: state.Name, Path: state.Root})
	}
	return
}

// Scan ищет skills/<package>/*/SKILL.md под installPrefix.
func Scan(installPrefix string, packageName string) (roots []Root, err error) {

	base := filepath.Join(installPrefix, "skills", packageName)
	var entries []os.DirEntry
	if entries, err = os.ReadDir(base); err != nil {
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
		skillPath := filepath.Join(base, entry.Name())
		if _, statErr := os.Stat(filepath.Join(skillPath, skillFileName)); statErr != nil {
			continue
		}
		roots = append(roots, Root{Name: entry.Name(), Path: skillPath})
	}
	return
}

// ResolveRoots выбирает roots из specs, иначе из states, иначе Scan.
func ResolveRoots(installPrefix string, packageName string, specs []models.SkillSpec, states []models.SkillState) (roots []Root, err error) {

	if roots = FromSpecs(installPrefix, specs); len(roots) > 0 {
		return
	}
	if roots = FromStates(states); len(roots) > 0 {
		return
	}
	return Scan(installPrefix, packageName)
}
