// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/pterm/pterm"
)

// getSubcommandPriority: чем меньше число, тем выше приоритет в списке.
func getSubcommandPriority(subcommandName string) (priority int) {

	priorityMap := map[string]int{
		"add":     1,
		"list":    2,
		"search":  3,
		"upgrade": 4,
		"del":     5,
		"info":    6,
		"repo":    7,
		"doc":     8,
		"update":  9,
		"init":    10,
	}

	var exists bool
	if priority, exists = priorityMap[subcommandName]; exists {
		return
	}

	// Команды, не в списке приоритетов, идут в конце
	priority = 999
	return
}

// PromptSubcommandSelection показывает интерактивный выбор подкоманды
func PromptSubcommandSelection(cmdPath []string, subcommands []Command) (selected Command) {

	if len(subcommands) == 0 {
		selected = nil
		return
	}

	// Сортируем команды по приоритету
	sort.Slice(subcommands, func(i, j int) bool {
		pathI := subcommands[i].GetPath()
		pathJ := subcommands[j].GetPath()

		subcommandNameI := ""
		if len(pathI) > 0 {
			lastElemI := pathI[len(pathI)-1]
			nameI, _ := parsePathElement(lastElemI)
			subcommandNameI = nameI
		}

		subcommandNameJ := ""
		if len(pathJ) > 0 {
			lastElemJ := pathJ[len(pathJ)-1]
			nameJ, _ := parsePathElement(lastElemJ)
			subcommandNameJ = nameJ
		}

		priorityI := getSubcommandPriority(subcommandNameI)
		priorityJ := getSubcommandPriority(subcommandNameJ)

		if priorityI != priorityJ {
			return priorityI < priorityJ
		}

		// Если приоритеты одинаковые, сортируем по имени
		return subcommandNameI < subcommandNameJ
	})

	// Формируем список опций для выбора
	options := make([]string, 0, len(subcommands))
	commandMap := make(map[string]Command)

	for _, cmd := range subcommands {
		path := cmd.GetPath()

		// Показываем только последний элемент пути (имя подкоманды)
		// и убираем алиасы (часть после ":")
		subcommandName := ""
		if len(path) > 0 {
			lastElem := path[len(path)-1]
			// Парсим элемент пути, извлекая только имя (без алиаса)
			name, _ := parsePathElement(lastElem)
			subcommandName = name
		}

		description := cmd.GetDescription()
		// Добавляем иконку для команды
		optionText := fmt.Sprintf("%s %s%s%s", iconCommand, subcommandName, optionSeparator, description)
		options = append(options, optionText)
		commandMap[optionText] = cmd
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(fmt.Sprintf(i18n.Msg("Select subcommand for %s"), strings.Join(cmdPath, " ")))

	if selectedOption == "" {
		selected = nil
		return
	}

	selected = commandMap[selectedOption]
	return
}

// PromptCommandSelection показывает единое меню выбора команд
func PromptCommandSelection() (selectedCommand string) {

	tree := GetCommandTree()
	if tree == nil {
		selectedCommand = ""
		return
	}

	// Собираем все корневые узлы (первый уровень) через виртуальный корневой узел
	rootNode := &CommandNode{
		children: tree.root,
	}
	rootChildNodes := rootNode.GetChildNodes()

	// Разделяем узлы на группы и команды
	groups := make([]*CommandNode, 0)
	commands := make([]*CommandNode, 0)

	for _, node := range rootChildNodes {
		if node.command != nil {
			// Узел с командой
			commands = append(commands, node)
		} else if len(node.children) > 0 {
			// Промежуточный узел с дочерними узлами (группа)
			groups = append(groups, node)
		}
	}

	// Сортируем группы и команды по имени
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].name < groups[j].name
	})
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].name < commands[j].name
	})

	// Формируем список опций: сначала группы, потом команды
	options := make([]string, 0, len(groups)+len(commands))
	nodeMap := make(map[string]*CommandNode)

	// Добавляем группы
	for _, node := range groups {
		optionText := fmt.Sprintf("%s %s", iconGroup, node.name)
		options = append(options, optionText)
		nodeMap[optionText] = node
	}

	// Добавляем команды
	for _, node := range commands {
		description := node.command.GetDescription()
		optionText := fmt.Sprintf("%s %s%s%s", iconCommand, node.name, optionSeparator, description)
		options = append(options, optionText)
		nodeMap[optionText] = node
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(i18n.Msg("Select command"))

	if selectedOption == "" {
		selectedCommand = ""
		return
	}

	var exists bool
	var selectedNode *CommandNode
	if selectedNode, exists = nodeMap[selectedOption]; !exists {
		selectedCommand = ""
		return
	}

	// Если у узла есть команда - возвращаем путь команды
	if selectedNode.command != nil {
		path := selectedNode.command.GetPath()
		normalizedPath := make([]string, 0, len(path))
		for _, elem := range path {
			name, _ := parsePathElement(elem)
			normalizedPath = append(normalizedPath, name)
		}
		selectedCommand = strings.Join(normalizedPath, " ")
		return
	}

	// Если у узла нет команды, но есть дочерние узлы - это промежуточный узел
	// Возвращаем имя узла, чтобы cobra мог обработать его как команду с подкомандами
	selectedCommand = selectedNode.name
	return
}
