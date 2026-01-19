// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"context"
	"errors"
	"sort"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/cli"
	"github.com/seniorGolang/tg/v3/internal/installer/models"

	"github.com/pterm/pterm"
)

func collectDocItems(ctx types.CommandContext) (items []*docItem, err error) {

	items = make([]*docItem, 0)

	var scope string
	if ctx.GlobalOpts.Scope != "" {
		scope = ctx.GlobalOpts.Scope
	}
	var inst *cli.Installer
	if inst, err = cli.NewInstaller(scope); err != nil {
		return
	}

	cmdCtx := context.Background()
	var installations []models.Installation
	if installations, err = inst.DatabaseManager().ListInstallations(cmdCtx); err != nil {
		return
	}

	for i := range installations {
		installation := &installations[i]

		packageName := installation.Package
		if installation.Source != "" {
			packageName = installation.Source + pathSeparator + installation.Package
		}
		var pkg *models.Package
		pkg, _, err = inst.ManifestManager().FindPackage(cmdCtx, packageName)
		if err != nil {
			continue
		}

		description := pkg.Descr
		if description == "" {
			description = i18n.Msg("Installed package")
		}

		var doc string
		var scopeForDoc string
		if ctx.GlobalOpts.Scope != "" {
			scopeForDoc = ctx.GlobalOpts.Scope
		}
		pluginDoc := getPluginDoc(ctx, scopeForDoc, pkg.Name, installation.Version)
		if pluginDoc != "" {
			doc = pluginDoc
		}

		items = append(items, &docItem{
			name:         pkg.Name,
			version:      installation.Version,
			description:  description,
			doc:          doc,
			installation: installation,
		})
	}

	return
}

func selectDocItemInteractive(items []*docItem, ctx types.CommandContext) (selectedItem *docItem, err error) {

	sort.Slice(items, func(i int, j int) (less bool) {
		less = items[i].name < items[j].name
		return
	})

	itemOptions := make([]string, 0, len(items))
	itemMap := make(map[string]*docItem)
	for _, item := range items {
		optionText := formatDocItemOption(item)
		itemOptions = append(itemOptions, optionText)
		itemMap[optionText] = item
	}

	selected, _ := pterm.DefaultInteractiveSelect.
		WithOptions(itemOptions).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(itemOptions))).
		Show(i18n.Msg("Select plugin to view documentation"))

	if selected == "" {
		return
	}

	var exists bool
	selectedItem, exists = itemMap[selected]
	if !exists {
		err = errors.New(i18n.Msg("Selected plugin not found"))
		return
	}

	return
}

func formatDocItemOption(item *docItem) (optionText string) {

	optionText = item.name
	if item.version != "" {
		optionText += docVersionSeparator + item.version
	}
	if item.description != "" {
		optionText += docDescriptionSeparator + item.description
	}
	return optionText
}

func createDocItemFromPackage(pkg *models.Package, manifest *models.Manifest, installation *models.Installation) (item *docItem) {

	var version string
	if installation != nil {
		version = installation.Version
	} else if manifest != nil {
		version = manifest.Version
	}

	return &docItem{
		name:         pkg.Name,
		version:      version,
		description:  pkg.Descr,
		doc:          "",
		installation: installation,
	}
}
