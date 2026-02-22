// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// checkCycles: DFS; recStack — узлы текущего пути, попадание в уже помеченный узел даёт цикл.
func (p *Planner) checkCycles(dependencyGraph map[string][]string) (err error) {

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cyclePath []string

	for node := range dependencyGraph {
		if !visited[node] {
			if p.hasCycleDFS(node, dependencyGraph, visited, recStack, &cyclePath) {
				cycleStr := ""
				for i, plugin := range cyclePath {
					if i > 0 {
						cycleStr += cyclePathSeparator
					}
					cycleStr += plugin
				}
				err = fmt.Errorf(i18n.Msg("detected circular dependency: %s"), cycleStr)
				return
			}
		}
	}

	return
}

func (p *Planner) hasCycleDFS(node string, graph map[string][]string, visited map[string]bool, recStack map[string]bool, cyclePath *[]string) (hasCycle bool) {

	visited[node] = true
	recStack[node] = true
	*cyclePath = append(*cyclePath, node)

	for _, dep := range graph[node] {
		if !visited[dep] {
			if p.hasCycleDFS(dep, graph, visited, recStack, cyclePath) {
				return true
			}
		} else if recStack[dep] {
			*cyclePath = append(*cyclePath, dep)
			return true
		}
	}

	recStack[node] = false
	*cyclePath = (*cyclePath)[:len(*cyclePath)-1]
	return false
}

func (p *Planner) validatePlan(allInstallations map[string]*models.Installation, dependencyGraph map[string][]string, commandPluginName string) (err error) {

	commandCount := 0
	for name, inst := range allInstallations {
		kind := detectKind(inst)
		if !isValidKind(kind) {
			return fmt.Errorf(i18n.Msg("plugin %s has invalid kind: %s"), name, kind)
		}

		if inst.Always {
			if kind != KindPre && kind != KindPost {
				return fmt.Errorf(i18n.Msg("plugin %s has always=true but kind is %s: always can only be used with pre or post plugins"), name, kind)
			}
		}

		if kind == KindCommand {
			commandCount++
			if name != commandPluginName {
				return fmt.Errorf(i18n.Msg("plugin %s is a command but not the requested command"), name)
			}
		}

		for _, dep := range dependencyGraph[name] {
			var depInst *models.Installation
			var exists bool
			if depInst, exists = allInstallations[dep]; !exists {
				continue
			}
			depKind := detectKind(depInst)
			if depKind != KindCommand {
				continue
			}
			// Pre/post могут объявлять зависимость от текущей команды (обратная привязка: «запускай меня при этой команде»).
			if dep == commandPluginName && (kind == KindPre || kind == KindPost) {
				continue
			}
			return fmt.Errorf(i18n.Msg("plugin %s cannot depend on command %s: commands cannot be dependencies of other plugins"), name, dep)
		}
	}

	if commandCount != 1 {
		return fmt.Errorf(i18n.Msg("plan must contain exactly one command, found %d"), commandCount)
	}

	return
}

func isValidKind(kind Kind) (valid bool) {
	return kind == KindPre || kind == KindStage || kind == KindCommand || kind == KindPost
}
