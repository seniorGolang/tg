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
		return errors.New(i18n.Msg("Field name is required"))
	}

	if pluginInfo.Version == "" {
		return errors.New(i18n.Msg("Field version is required"))
	}

	if pluginInfo.Description == "" {
		return errors.New(i18n.Msg("Field description is required"))
	}

	if pluginInfo.Author == "" {
		return errors.New(i18n.Msg("Field author is required"))
	}

	if pluginInfo.License == "" {
		return errors.New(i18n.Msg("Field license is required"))
	}

	if expectedName != "" && pluginInfo.Name != expectedName {
		return fmt.Errorf(i18n.Msg("Plugin name mismatch: expected %s, got %s"), expectedName, pluginInfo.Name)
	}

	if expectedVersion != "" && pluginInfo.Version != expectedVersion {
		return fmt.Errorf(i18n.Msg("Plugin version mismatch: expected %s, got %s"), expectedVersion, pluginInfo.Version)
	}

	return
}
