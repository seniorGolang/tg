// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

// optimizePlan убирает дубликаты шагов с одинаковым именем и версией (один плагин может попасть в несколько групп: pre, stage, post).
func (p *Planner) optimizePlan(steps []Step) (optimized []Step) {

	seen := make(map[string]bool)
	optimized = make([]Step, 0, len(steps))

	for _, step := range steps {
		key := step.Name + dependencyVersionSeparator + step.PluginVersion
		if !seen[key] {
			seen[key] = true
			optimized = append(optimized, step)
		}
	}

	return
}
