// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package validator

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func ValidateMetadata(pluginInfo plugin.Info, expectedName string, expectedVersion string) (err error) {

	if pluginInfo.Name == "" {
		err = errors.New(i18n.Msg("Field name is required"))
		return
	}

	if pluginInfo.Version == "" {
		err = errors.New(i18n.Msg("Field version is required"))
		return
	}

	if pluginInfo.Description == "" {
		err = errors.New(i18n.Msg("Field description is required"))
		return
	}

	if pluginInfo.Author == "" {
		err = errors.New(i18n.Msg("Field author is required"))
		return
	}

	if pluginInfo.License == "" {
		err = errors.New(i18n.Msg("Field license is required"))
		return
	}

	if expectedName != "" && pluginInfo.Name != expectedName {
		err = fmt.Errorf(i18n.Msg("Plugin name mismatch: expected %s, got %s"), expectedName, pluginInfo.Name)
		return
	}

	if expectedVersion != "" && pluginInfo.Version != expectedVersion {
		err = fmt.Errorf(i18n.Msg("Plugin version mismatch: expected %s, got %s"), expectedVersion, pluginInfo.Version)
		return
	}

	return
}
