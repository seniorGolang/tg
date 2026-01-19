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

// HandleList обрабатывает команду list.
func (inst *Installer) HandleList(ctx context.Context, args []string) (err error) {

	// Проверяем, был ли scope переопределен через --scope
	// Если inst.currentScope отличается от текущего активного scope в хранилище,
	// значит scope был переопределен и нужно показывать только его
	activeScope, _ := inst.scopeManager.GetCurrentScope(ctx)
	scopeOverridden := inst.currentScope != activeScope

	// Структура для группировки: scope -> source -> package -> []Installation
	type scopeData struct {
		sources map[string]map[string][]models.Installation
	}

	allData := make(map[string]*scopeData)

	var scopesToProcess []string
	if scopeOverridden {
		// Если scope переопределен, обрабатываем только указанный scope
		scopesToProcess = []string{inst.currentScope}
	} else {
		// Получаем все scopes
		var scopes []models.ScopeInfo
		if scopes, err = inst.scopeManager.ListScopes(ctx); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to get list of scopes: %w"), err)
			return
		}
		scopesToProcess = make([]string, 0, len(scopes))
		for _, scopeInfo := range scopes {
			scopesToProcess = append(scopesToProcess, scopeInfo.Name)
		}
	}

	// Собираем установки из указанных scopes
	for _, scopeName := range scopesToProcess {
		allData[scopeName] = &scopeData{
			sources: make(map[string]map[string][]models.Installation),
		}

		// Читаем установки для этого scope
		dbFile := storage.GetPackagesDBFile(scopeName)
		var statErr error
		if _, statErr = os.Stat(dbFile); os.IsNotExist(statErr) {
			continue
		}

		var data []byte
		if data, err = os.ReadFile(dbFile); err != nil {
			continue
		}

		var db models.InstallationDatabase
		//nolint:musttag // структура InstallationDatabase и все вложенные структуры имеют теги yaml в пакете models
		if err = yaml.Unmarshal(data, &db); err != nil {
			continue
		}

		// Группируем по source -> package
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

	// Сортируем scopes
	scopeNames := make([]string, 0, len(allData))
	for scopeName := range allData {
		scopeNames = append(scopeNames, scopeName)
	}
	sort.Strings(scopeNames)

	// Если scope переопределен и пуст, выводим сообщение
	if scopeOverridden && len(scopeNames) == 1 {
		pterm.Info.Printf(i18n.Msg("Scope %s is empty (no packages installed)")+"\n", inst.currentScope)
		return
	}

	// Строим дерево для pterm
	scopeNodes := make([]pterm.TreeNode, 0, len(scopeNames))

	for _, scopeName := range scopeNames {
		scopeData := allData[scopeName]

		// Сортируем источники
		sourceNames := make([]string, 0, len(scopeData.sources))
		for sourceName := range scopeData.sources {
			sourceNames = append(sourceNames, sourceName)
		}
		sort.Strings(sourceNames)

		// Строим узлы источников
		sourceNodes := make([]pterm.TreeNode, 0, len(sourceNames))

		for _, sourceName := range sourceNames {
			packages := scopeData.sources[sourceName]

			// Сортируем пакеты
			packageNames := make([]string, 0, len(packages))
			for packageName := range packages {
				packageNames = append(packageNames, packageName)
			}
			sort.Strings(packageNames)

			// Строим узлы пакетов
			packageNodes := make([]pterm.TreeNode, 0, len(packageNames))

			for _, packageName := range packageNames {
				installations := packages[packageName]

				// Строим узлы версий
				versionNodes := make([]pterm.TreeNode, 0, len(installations))

				for _, installation := range installations {
					// Используем сохраненное описание
					descr := installation.Descr

					// Форматируем версию в едином формате (v2.4.14)
					versionStr := installation.Version
					if versionStr != "" && !strings.HasPrefix(versionStr, ver.VersionPrefix) {
						versionStr = ver.VersionPrefix + versionStr
					}

					// Формируем текст версии
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

	// Выводим дерево через pterm
	for i, scopeNode := range scopeNodes {
		if err = pterm.DefaultTree.WithRoot(scopeNode).Render(); err != nil {
			return
		}
		// Добавляем пустую строку между деревьями, если их несколько
		if i < len(scopeNodes)-1 {
			fmt.Println()
		}
	}

	return
}
