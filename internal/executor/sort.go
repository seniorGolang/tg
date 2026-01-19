// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

// topologicalSort выполняет топологическую сортировку плагинов по графу зависимостей (алгоритм Кана).
// Плагины без зависимостей в списке идут первыми.
func (p *Planner) topologicalSort(plugins []string, dependencyGraph map[string][]string) (sorted []string) {

	inDegree := make(map[string]int)
	for _, name := range plugins {
		inDegree[name] = 0
	}

	for _, name := range plugins {
		for _, dep := range dependencyGraph[name] {
			var exists bool
			for _, p := range plugins {
				if p == dep {
					exists = true
					break
				}
			}
			if exists {
				inDegree[name]++
			}
		}
	}

	queue := make([]string, 0)
	for _, name := range plugins {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	sorted = make([]string, 0, len(plugins))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, name := range plugins {
			if name == current {
				continue
			}
			for _, dep := range dependencyGraph[name] {
				if dep == current {
					inDegree[name]--
					if inDegree[name] == 0 {
						queue = append(queue, name)
					}
					break
				}
			}
		}
	}

	return
}
