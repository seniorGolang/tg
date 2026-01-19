// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"

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

func (p *Planner) Plan(commandPluginName string, initialRequest plugin.Storage, rootDir string, commandPath []string, commandArgs []string) (plan Plan, err error) {

	var commandInstallation *models.Installation
	if commandInstallation, err = p.loader.GetInfo(commandPluginName); err != nil {
		err = fmt.Errorf(i18n.Msg("command %s not found: %w"), commandPluginName, err)
		return
	}

	if len(commandInstallation.Commands) == 0 {
		err = fmt.Errorf(i18n.Msg("plugin %s is not a command"), commandPluginName)
		return
	}

	allInstallations := make(map[string]*models.Installation)
	dependencyGraph := make(map[string][]string)
	dependencySpecs := make(map[string][]string)

	allInstallations[commandPluginName] = commandInstallation

	var alwaysPrePlugins map[string]*models.Installation
	var alwaysPostPlugins map[string]*models.Installation
	if alwaysPrePlugins, alwaysPostPlugins, err = p.collectAlwaysPlugins(); err != nil {
		return
	}

	for name, inst := range alwaysPrePlugins {
		allInstallations[name] = inst
	}
	for name, inst := range alwaysPostPlugins {
		allInstallations[name] = inst
	}

	if err = p.collectDirectChain(commandPluginName, allInstallations, dependencyGraph, dependencySpecs); err != nil {
		return
	}

	if err = p.collectAlwaysDependencies(alwaysPrePlugins, allInstallations, dependencyGraph, dependencySpecs); err != nil {
		return
	}

	if err = p.collectAlwaysDependencies(alwaysPostPlugins, allInstallations, dependencyGraph, dependencySpecs); err != nil {
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

	var steps []Step
	if steps, err = p.buildSteps(allInstallations, dependencyGraph, commandPluginName, alwaysPrePlugins, alwaysPostPlugins); err != nil {
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
