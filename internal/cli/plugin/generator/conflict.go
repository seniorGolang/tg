// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"log/slog"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func ResolveConflictsForNames(loader pluginLoader, pluginNames []string) (validPluginNames []string) {

	if len(pluginNames) == 0 {
		return
	}

	pathOwners := make(map[string]string)
	validNames := make([]string, 0)

	for _, pluginName := range pluginNames {
		var installation *models.Installation
		var err error
		if installation, err = loader.GetInfo(pluginName); err != nil {
			continue
		}

		if len(installation.InitPkgs) == 0 {
			continue
		}

		hasConflict := checkConflictsForNames(pluginName, installation, pathOwners)

		if !hasConflict {
			validNames = append(validNames, pluginName)
			registerPathsForNames(pluginName, installation, pathOwners)
		}
	}

	return validNames
}

func checkConflictsForNames(pluginName string, installation *models.Installation, pathOwners map[string]string) (hasConflict bool) {

	for _, pkg := range installation.InitPkgs {
		pathKey := "@root/" + pkg
		var owner string
		var exists bool
		if owner, exists = pathOwners[pathKey]; exists {
			slog.Warn(i18n.Msg("Path conflict: plugin skipped"),
				"plugin", pluginName,
				"version", installation.Version,
				"path", pathKey,
				"conflicting_plugin", owner)
			hasConflict = true
			return
		}
	}

	return
}

func registerPathsForNames(pluginName string, installation *models.Installation, pathOwners map[string]string) {

	for _, pkg := range installation.InitPkgs {
		pathKey := "@root/" + pkg
		pathOwners[pathKey] = pluginName
	}
}
