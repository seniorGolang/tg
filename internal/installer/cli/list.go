// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	ver "github.com/seniorGolang/tg/v3/internal/installer/version"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

func (inst *Installer) HandleList(ctx context.Context, args []string) (err error) {

	activeScope, _ := inst.scopeManager.GetCurrentScope(ctx)
	scopeOverridden := inst.currentScope != activeScope

	// Структура для группировки: scope -> source -> package -> []Installation
	type scopeData struct {
		sources map[string]map[string][]models.Installation
	}

	allData := make(map[string]*scopeData)

	var scopesToProcess []string
	if scopeOverridden {
		scopesToProcess = []string{inst.currentScope}
	} else {
		scopesToProcess = []string{activeScope}
	}

	for _, scopeName := range scopesToProcess {
		allData[scopeName] = &scopeData{
			sources: make(map[string]map[string][]models.Installation),
		}

		dbFile := storage.GetPackagesDBFile(scopeName)
		var data []byte
		var statErr error
		if _, statErr = os.Stat(dbFile); os.IsNotExist(statErr) {
			continue
		}

		if data, err = os.ReadFile(dbFile); err != nil {
			continue
		}

		var db models.InstallationDatabase
		//nolint:musttag // структура InstallationDatabase и все вложенные структуры имеют теги yaml в пакете models
		if err = yaml.Unmarshal(data, &db); err != nil {
			continue
		}

		for _, installation := range db.Installed {
			source := installation.Source
			if source == "" {
				source = i18n.Msg("(no source)")
			}

			if allData[scopeName].sources[source] == nil {
				allData[scopeName].sources[source] = make(map[string][]models.Installation)
			}

			allData[scopeName].sources[source][installation.Package] = append(
				allData[scopeName].sources[source][installation.Package],
				installation,
			)
		}
	}

	scopeNames := make([]string, 0, len(allData))
	for scopeName := range allData {
		scopeNames = append(scopeNames, scopeName)
	}
	sort.Strings(scopeNames)

	if scopeOverridden && len(scopeNames) == 1 {
		pterm.Info.Printf(i18n.Msg("Scope %s is empty (no packages installed)")+"\n", inst.currentScope)
		return
	}

	scopeNodes := make([]pterm.TreeNode, 0, len(scopeNames))

	for _, scopeName := range scopeNames {
		sd := allData[scopeName]

		sourceNames := make([]string, 0, len(sd.sources))
		for sourceName := range sd.sources {
			sourceNames = append(sourceNames, sourceName)
		}
		sort.Strings(sourceNames)

		sourceNodes := make([]pterm.TreeNode, 0, len(sourceNames))

		for _, sourceName := range sourceNames {
			packages := sd.sources[sourceName]

			packageNames := make([]string, 0, len(packages))
			for packageName := range packages {
				packageNames = append(packageNames, packageName)
			}
			sort.Strings(packageNames)

			packageNodes := make([]pterm.TreeNode, 0, len(packageNames))

			for _, packageName := range packageNames {
				installations := packages[packageName]

				versionNodes := make([]pterm.TreeNode, 0, len(installations))

				for _, installation := range installations {
					descr := installation.Descr

					versionStr := installation.Version
					if versionStr != "" && !strings.HasPrefix(versionStr, ver.VersionPrefix) {
						versionStr = ver.VersionPrefix + versionStr
					}

					versionText := versionStr
					if descr != "" {
						versionText = fmt.Sprintf("%s - %s", versionStr, descr)
					}

					versionNodes = append(versionNodes, pterm.TreeNode{
						Text: versionText,
					})
				}

				packageNodes = append(packageNodes, pterm.TreeNode{
					Text:     packageName,
					Children: versionNodes,
				})
			}

			sourceNodes = append(sourceNodes, pterm.TreeNode{
				Text:     sourceName,
				Children: packageNodes,
			})
		}

		scopeNodes = append(scopeNodes, pterm.TreeNode{
			Text:     scopeName,
			Children: sourceNodes,
		})
	}

	for i, scopeNode := range scopeNodes {
		if err = pterm.DefaultTree.WithRoot(scopeNode).Render(); err != nil {
			return
		}
		if i < len(scopeNodes)-1 {
			fmt.Println()
		}
	}

	return
}
