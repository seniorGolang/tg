// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func (p *Planner) collectAlwaysPlugins() (alwaysPre map[string]*models.Installation, alwaysPost map[string]*models.Installation, err error) {

	var listErr error
	var allInstallations []models.Installation
	if allInstallations, listErr = p.loader.GetList(); listErr != nil {
		err = fmt.Errorf(i18n.Msg("failed to list plugins: %w"), listErr)
		return
	}

	alwaysPre = make(map[string]*models.Installation)
	alwaysPost = make(map[string]*models.Installation)
	for i := range allInstallations {
		inst := &allInstallations[i]
		if !inst.Always {
			continue
		}

		kind := detectKind(inst)
		if kind != KindPre && kind != KindPost {
			err = fmt.Errorf(i18n.Msg("plugin %s has always=true but kind is %s: always can only be used with pre or post plugins"), inst.Package, kind)
			return
		}

		switch kind {
		case KindPre:
			alwaysPre[inst.Package] = inst
		case KindPost:
			alwaysPost[inst.Package] = inst
		}
	}

	return
}

func (p *Planner) collectDirectChain(pluginName string, allInstallations map[string]*models.Installation, dependencyGraph map[string][]string, dependencySpecs map[string][]string) (err error) {

	var inst *models.Installation
	var exists bool
	inst, exists = allInstallations[pluginName]
	if !exists {
		if inst, err = p.loader.GetInfo(pluginName); err != nil {
			err = fmt.Errorf(i18n.Msg("plugin %s not found: %w"), pluginName, err)
			return
		}
		allInstallations[pluginName] = inst
	}

	dependencies := inst.Dependencies
	dependencySpecs[pluginName] = dependencies

	parsedDeps := make([]string, 0)
	for _, dep := range dependencies {
		var depName string
		if depName, _, err = ParseDependency(dep); err != nil {
			err = fmt.Errorf(i18n.Msg("error parsing dependency %s: %w"), dep, err)
			return
		}
		parsedDeps = append(parsedDeps, depName)

		if _, exists = allInstallations[depName]; !exists {
			if err = p.collectDirectChain(depName, allInstallations, dependencyGraph, dependencySpecs); err != nil {
				return
			}
		}
	}

	dependencyGraph[pluginName] = parsedDeps
	return
}

func (p *Planner) collectAlwaysDependencies(alwaysPlugins map[string]*models.Installation, allInstallations map[string]*models.Installation, dependencyGraph map[string][]string, dependencySpecs map[string][]string) (err error) {

	for pluginName := range alwaysPlugins {
		if err = p.collectDirectChain(pluginName, allInstallations, dependencyGraph, dependencySpecs); err != nil {
			return
		}
	}

	return
}

func (p *Planner) resolveVersions(allInstallations map[string]*models.Installation, dependencySpecs map[string][]string) (err error) {

	for pluginName, deps := range dependencySpecs {
		for _, depSpec := range deps {
			var depName string
			var depVersion string
			if depName, depVersion, err = ParseDependency(depSpec); err != nil {
				err = fmt.Errorf(i18n.Msg("error parsing dependency %s: %w"), depSpec, err)
				return
			}

			var depInstallation *models.Installation
			if depInstallation, err = p.loader.GetInfo(depName); err != nil {
				err = fmt.Errorf(i18n.Msg("dependency %s of plugin %s is not installed, install it via plugin install command"), depName, pluginName)
				return
			}

			if depVersion != "" {
				installedVersion := depInstallation.Version
				if !isVersionCompatible(installedVersion, depVersion) {
					err = fmt.Errorf(i18n.Msg("version of plugin %s (%s) is not compatible with requirement %s of plugin %s, install compatible version via plugin install command"), depName, installedVersion, depVersion, pluginName)
					return
				}
			}

			allInstallations[depName] = depInstallation
		}
	}

	return
}

func (p *Planner) collectDirectChainSet(pluginName string, dependencyGraph map[string][]string, resultSet map[string]bool) {

	resultSet[pluginName] = true
	for _, dep := range dependencyGraph[pluginName] {
		if !resultSet[dep] {
			p.collectDirectChainSet(dep, dependencyGraph, resultSet)
		}
	}
}

// collectCommandBoundPrePost: обратная зависимость — плагин зависит от команды => включаем плагин при запуске этой команды.
func (p *Planner) collectCommandBoundPrePost(allInstallations map[string]*models.Installation, dependencyGraph map[string][]string, dependencySpecs map[string][]string, directChainSet map[string]bool) (err error) {

	var listErr error
	var allList []models.Installation
	if allList, listErr = p.loader.GetList(); listErr != nil {
		err = fmt.Errorf(i18n.Msg("failed to list plugins: %w"), listErr)
		return
	}

	for i := range allList {
		inst := &allList[i]
		if allInstallations[inst.Package] != nil {
			continue
		}

		kind := detectKind(inst)
		if kind != KindPre && kind != KindPost {
			continue
		}

		for _, depSpec := range inst.Dependencies {
			var depName string
			if depName, _, err = ParseDependency(depSpec); err != nil {
				err = fmt.Errorf(i18n.Msg("error parsing dependency %s: %w"), depSpec, err)
				return
			}
			if !directChainSet[depName] {
				continue
			}
			if err = p.collectDirectChain(inst.Package, allInstallations, dependencyGraph, dependencySpecs); err != nil {
				return
			}
			break
		}
	}

	return
}
