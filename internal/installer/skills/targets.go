// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// Target — корневой каталог инструмента и подкаталог skills внутри него.
type Target struct {
	Name   string
	Root   string
	Skills string
}

// ResolveTargets строит список targets относительно home.
// Каталог Root должен существовать, иначе target пропускается (если mkdir=false)
// или создаётся (если mkdir=true).
func ResolveTargets(home string, names []string, mkdir bool) (targets []Target, skipped []string, err error) {

	targets = make([]Target, 0, len(names))
	skipped = make([]string, 0)

	for _, name := range names {
		var root string
		switch name {
		case TargetAgents:
			root = filepath.Join(home, ".agents")
		case TargetCursor:
			root = filepath.Join(home, ".cursor")
		case TargetClaude:
			root = filepath.Join(home, ".claude")
		case TargetCodex:
			root = filepath.Join(home, ".codex")
		default:
			return nil, nil, fmt.Errorf("unknown skills target: %s", name)
		}

		var info os.FileInfo
		if info, err = os.Stat(root); err != nil {
			if !os.IsNotExist(err) {
				return nil, nil, err
			}
			err = nil
			if !mkdir {
				skipped = append(skipped, name)
				continue
			}
			if err = os.MkdirAll(root, 0755); err != nil {
				return nil, nil, err
			}
		} else if !info.IsDir() {
			return nil, nil, fmt.Errorf("skills target root is not a directory: %s", root)
		}

		skillsDir := filepath.Join(root, "skills")
		targets = append(targets, Target{
			Name:   name,
			Root:   root,
			Skills: skillsDir,
		})
	}

	return
}
