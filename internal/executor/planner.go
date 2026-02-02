// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"
	"slices"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

type Planner struct {
	loader pluginLoader
}

func NewPlanner(loader pluginLoader) (planner *Planner) {

	return &Planner{loader: loader}
}

func (p *Planner) collectPlanData(commandPluginName string) (allInstallations map[string]*models.Installation, dependencyGraph map[string][]string, alwaysPre map[string]*models.Installation, alwaysPost map[string]*models.Installation, err error) {

	var commandInstallation *models.Installation
	if commandInstallation, err = p.loader.GetInfo(commandPluginName); err != nil {
		err = fmt.Errorf(i18n.Msg("command %s not found: %w"), commandPluginName, err)
		return
	}

	if len(commandInstallation.Commands) == 0 {
		err = fmt.Errorf(i18n.Msg("plugin %s is not a command"), commandPluginName)
		return
	}

	allInstallations = make(map[string]*models.Installation)
	dependencyGraph = make(map[string][]string)
	dependencySpecs := make(map[string][]string)

	allInstallations[commandPluginName] = commandInstallation

	if alwaysPre, alwaysPost, err = p.collectAlwaysPlugins(); err != nil {
		return
	}

	for name, inst := range alwaysPre {
		allInstallations[name] = inst
	}
	for name, inst := range alwaysPost {
		allInstallations[name] = inst
	}

	if err = p.collectDirectChain(commandPluginName, allInstallations, dependencyGraph, dependencySpecs); err != nil {
		return
	}

	if err = p.collectAlwaysDependencies(alwaysPre, allInstallations, dependencyGraph, dependencySpecs); err != nil {
		return
	}

	if err = p.collectAlwaysDependencies(alwaysPost, allInstallations, dependencyGraph, dependencySpecs); err != nil {
		return
	}

	directChainSet := make(map[string]bool)
	p.collectDirectChainSet(commandPluginName, dependencyGraph, directChainSet)
	if err = p.collectCommandBoundPrePost(allInstallations, dependencyGraph, dependencySpecs, directChainSet); err != nil {
		return
	}

	if err = p.resolveVersions(allInstallations, dependencySpecs); err != nil {
		return
	}

	if err = p.checkCycles(dependencyGraph); err != nil {
		return
	}

	if err = p.validatePlan(allInstallations, dependencyGraph, commandPluginName); err != nil {
		return
	}

	return
}

func (p *Planner) Plan(commandPluginName string, initialRequest plugin.Storage, rootDir string, commandPath []string, commandArgs []string) (plan Plan, err error) {

	var allInstallations map[string]*models.Installation
	var dependencyGraph map[string][]string
	var alwaysPre map[string]*models.Installation
	var alwaysPost map[string]*models.Installation
	if allInstallations, dependencyGraph, alwaysPre, alwaysPost, err = p.collectPlanData(commandPluginName); err != nil {
		return
	}

	var steps []Step
	if steps, err = p.buildSteps(allInstallations, dependencyGraph, commandPluginName, alwaysPre, alwaysPost); err != nil {
		return
	}

	plan = Plan{
		Steps:          steps,
		InitialRequest: initialRequest,
		RootDir:        rootDir,
		CommandPath:    commandPath,
		CommandArgs:    commandArgs,
		StepKeys:       make(map[string]stepKeysInfo),
	}

	return
}

func (p *Planner) GetMergedOptionsForCommand(commandPluginName string, commandPath []string) (options []models.OptionInfo, err error) {

	var allInstallations map[string]*models.Installation
	var dependencyGraph map[string][]string
	var alwaysPre map[string]*models.Installation
	var alwaysPost map[string]*models.Installation
	if allInstallations, dependencyGraph, alwaysPre, alwaysPost, err = p.collectPlanData(commandPluginName); err != nil {
		return
	}

	var steps []Step
	if steps, err = p.buildSteps(allInstallations, dependencyGraph, commandPluginName, alwaysPre, alwaysPost); err != nil {
		return
	}

	options = p.mergeOptionsFromSteps(steps, allInstallations, commandPluginName, commandPath)
	return
}

func (p *Planner) mergeOptionsFromSteps(steps []Step, allInstallations map[string]*models.Installation, commandPluginName string, commandPath []string) (result []models.OptionInfo) {

	occupied := make(map[string]bool)
	result = make([]models.OptionInfo, 0)

	for i := range steps {
		step := &steps[i]
		if step.Kind != KindCommand {
			continue
		}
		inst := allInstallations[commandPluginName]
		if inst == nil {
			break
		}
		cmdOptions := p.getCommandOptions(inst, commandPath)
		for _, opt := range cmdOptions {
			if !occupied[opt.Name] {
				occupied[opt.Name] = true
				result = append(result, opt)
			}
		}
		break
	}

	for i := range steps {
		step := &steps[i]
		if step.Kind == KindCommand {
			continue
		}
		inst := allInstallations[step.Name]
		if inst == nil || len(inst.Options) == 0 {
			continue
		}
		for _, opt := range inst.Options {
			if !occupied[opt.Name] {
				occupied[opt.Name] = true
				result = append(result, opt)
			}
		}
	}

	return
}

func (p *Planner) getCommandOptions(inst *models.Installation, commandPath []string) (opts []models.OptionInfo) {

	for i := range inst.Commands {
		cmd := &inst.Commands[i]
		if slices.Equal(cmd.Path, commandPath) {
			if len(cmd.Options) > 0 {
				return cmd.Options
			}
			break
		}
	}

	return inst.Options
}
