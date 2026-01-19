// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func (p *Planner) buildSteps(allInstallations map[string]*models.Installation, dependencyGraph map[string][]string, commandPluginName string, alwaysPre map[string]*models.Installation, alwaysPost map[string]*models.Installation) (steps []Step, err error) {

	directChainSet := make(map[string]bool)
	p.collectDirectChainSet(commandPluginName, dependencyGraph, directChainSet)

	alwaysPreSet := make(map[string]bool)
	for name := range alwaysPre {
		alwaysPreSet[name] = true
	}
	for name := range alwaysPre {
		p.collectDirectChainSet(name, dependencyGraph, alwaysPreSet)
	}

	alwaysPostSet := make(map[string]bool)
	for name := range alwaysPost {
		alwaysPostSet[name] = true
	}
	for name := range alwaysPost {
		p.collectDirectChainSet(name, dependencyGraph, alwaysPostSet)
	}

	// Группы: preAlways/postAlways — плагины с always=true; preChain/stageChain/postChain — цепочка зависимостей команды.
	// Итоговый порядок шагов: preAlways → preChain → stageChain → command → postChain → postAlways.
	var preAlways []string
	var preChain []string
	var stageChain []string
	var postChain []string
	var postAlways []string

	for name, inst := range allInstallations {
		if name == commandPluginName {
			continue
		}

		kind := detectKind(inst)
		inDirectChain := directChainSet[name]
		inAlwaysPre := alwaysPreSet[name]
		inAlwaysPost := alwaysPostSet[name]

		switch {
		case inDirectChain:
			switch kind {
			case KindPre:
				preChain = append(preChain, name)
			case KindStage:
				stageChain = append(stageChain, name)
			case KindPost:
				postChain = append(postChain, name)
			}
		case inAlwaysPre && !inDirectChain:
			switch kind {
			case KindPre:
				preAlways = append(preAlways, name)
			case KindStage:
				stageChain = append(stageChain, name)
			case KindPost:
				postChain = append(postChain, name)
			}
		case inAlwaysPost && !inDirectChain:
			switch kind {
			case KindPost:
				postAlways = append(postAlways, name)
			case KindStage:
				stageChain = append(stageChain, name)
			case KindPre:
				preChain = append(preChain, name)
			}
		}
	}

	preAlwaysSorted := p.topologicalSort(preAlways, dependencyGraph)
	preChainSorted := p.topologicalSort(preChain, dependencyGraph)
	stageChainSorted := p.topologicalSort(stageChain, dependencyGraph)
	postChainSorted := p.topologicalSort(postChain, dependencyGraph)
	postAlwaysSorted := p.topologicalSort(postAlways, dependencyGraph)

	steps = make([]Step, 0)

	for _, name := range preAlwaysSorted {
		inst := allInstallations[name]
		steps = append(steps, Step{
			Name:          name,
			PluginVersion: inst.Version,
			Kind:          KindPre,
			Dependencies:  dependencyGraph[name],
			Silent:        inst.Silent,
		})
	}

	for _, name := range preChainSorted {
		inst := allInstallations[name]
		steps = append(steps, Step{
			Name:          name,
			PluginVersion: inst.Version,
			Kind:          KindPre,
			Dependencies:  dependencyGraph[name],
			Silent:        inst.Silent,
		})
	}

	for _, name := range stageChainSorted {
		inst := allInstallations[name]
		steps = append(steps, Step{
			Name:          name,
			PluginVersion: inst.Version,
			Kind:          KindStage,
			Dependencies:  dependencyGraph[name],
			Silent:        inst.Silent,
		})
	}

	commandInst := allInstallations[commandPluginName]
	steps = append(steps, Step{
		Name:          commandPluginName,
		PluginVersion: commandInst.Version,
		Kind:          KindCommand,
		Dependencies:  dependencyGraph[commandPluginName],
		Silent:        commandInst.Silent,
	})

	for _, name := range postChainSorted {
		inst := allInstallations[name]
		steps = append(steps, Step{
			Name:          name,
			PluginVersion: inst.Version,
			Kind:          KindPost,
			Dependencies:  dependencyGraph[name],
			Silent:        inst.Silent,
		})
	}

	for _, name := range postAlwaysSorted {
		inst := allInstallations[name]
		steps = append(steps, Step{
			Name:          name,
			PluginVersion: inst.Version,
			Kind:          KindPost,
			Dependencies:  dependencyGraph[name],
			Silent:        inst.Silent,
		})
	}

	steps = p.optimizePlan(steps)

	return
}
