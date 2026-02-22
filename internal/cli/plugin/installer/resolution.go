// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installer

import (
	"context"
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// resolveVersion разрешает версию плагина, если она не указана.
func (i *PluginInstaller) resolveVersion(ctx context.Context, pluginName string, version string) (resolvedPluginName string, resolvedVersion string, err error) {

	if version != "" {
		resolvedPluginName = pluginName
		resolvedVersion = version
		return
	}

	if pluginName != "" {
		var versions []VersionInfo
		if versions, err = i.source.ListVersions(ctx, pluginName); err != nil {
			return "", "", fmt.Errorf(i18n.Msg("Failed to get list of versions: %w"), err)
		}

		if len(versions) == 0 {
			return "", "", fmt.Errorf(i18n.Msg("Plugin %s versions not found"), pluginName)
		}

		resolvedPluginName = pluginName
		resolvedVersion = versions[0].Version
		return
	}

	var plugins []PluginInfo
	if plugins, err = i.source.ListPlugins(ctx); err != nil {
		return "", "", fmt.Errorf(i18n.Msg("Failed to get list of plugins: %w"), err)
	}

	if len(plugins) == 0 {
		return "", "", errors.New(i18n.Msg("No plugins found in repository"))
	}

	resolvedPluginName = plugins[0].Name
	if len(plugins[0].Versions) == 0 {
		return "", "", fmt.Errorf(i18n.Msg("Plugin %s versions not found"), resolvedPluginName)
	}

	return plugins[0].Name, plugins[0].Versions[0].Version, nil
}
