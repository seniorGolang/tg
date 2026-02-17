// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

// optimizePlan убирает дубликаты шагов по имени плагина (один плагин может попасть в несколько групп: pre, stage, post).
// Остаётся первое по порядку вхождение шага с данным именем; последующие с тем же именем отбрасываются.
func (p *Planner) optimizePlan(steps []Step) (optimized []Step) {

	seen := make(map[string]bool)
	optimized = make([]Step, 0, len(steps))

	for _, step := range steps {
		if !seen[step.Name] {
			seen[step.Name] = true
			optimized = append(optimized, step)
		}
	}

	return
}
