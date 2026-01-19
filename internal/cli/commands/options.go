// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// extractOptionsFromFlags извлекает опции из флагов команды (исключая позиционные)
func extractOptionsFromFlags(cobraCmd *cobra.Command, commandOptions []Option) (options map[string]any) {

	options = make(map[string]any)

	for _, opt := range commandOptions {
		// Пропускаем позиционные опции - они обрабатываются отдельно
		if opt.IsPositional {
			continue
		}

		// Проверяем, был ли флаг реально задан пользователем
		wasChanged := cobraCmd.Flags().Changed(opt.Name)

		switch opt.Type {
		case optionTypeString:
			val, _ := cobraCmd.Flags().GetString(opt.Name)
			// Добавляем опцию только если:
			// 1. Флаг был задан пользователем (wasChanged), или
			// 2. Есть значение по умолчанию (opt.Default != nil)
			if wasChanged {
				options[opt.Name] = val
			} else if opt.Default != nil {
				options[opt.Name] = opt.Default
			}
		case optionTypeInt:
			val, _ := cobraCmd.Flags().GetInt(opt.Name)
			if wasChanged {
				options[opt.Name] = val
			} else if opt.Default != nil {
				options[opt.Name] = opt.Default
			}
		case optionTypeBool:
			val, _ := cobraCmd.Flags().GetBool(opt.Name)
			// Для bool всегда добавляем значение, так как оно имеет явное значение по умолчанию
			options[opt.Name] = val
		}
	}

	return
}

func getNonPositionalOptions(options []Option) (nonPositionalOptions []Option) {

	nonPositionalOptions = make([]Option, 0)
	for _, opt := range options {
		if !opt.IsPositional {
			nonPositionalOptions = append(nonPositionalOptions, opt)
		}
	}
	return
}

func getPositionalOptions(options []Option) (positionalOptions []Option) {

	positionalOptions = make([]Option, 0)
	for _, opt := range options {
		if opt.IsPositional {
			positionalOptions = append(positionalOptions, opt)
		}
	}
	return
}

// PromptCommandOptions запрашивает опции для любой команды
func PromptCommandOptions(cmd Command, currentOptions map[string]any, commandPath []string) (options map[string]any) {

	commandOptions := cmd.GetOptions()

	// Если опций нет, возвращаем текущие опции
	if len(commandOptions) == 0 {
		options = currentOptions
		return
	}

	// Специальная логика для команды plugin init
	if len(commandPath) == 2 && commandPath[0] == "plugin" && commandPath[1] == "init" {
		return promptPluginInitOptions(commandOptions, currentOptions, commandPath)
	}

	return promptOptions(commandOptions, currentOptions)
}

// promptOptions запрашивает опции интерактивно
func promptOptions(commandOptions []Option, currentOptions map[string]any) (options map[string]any) {

	// Запрашиваем каждую опцию интерактивно
	options = make(map[string]any)

	for _, opt := range commandOptions {
		currentVal, hasCurrent := currentOptions[opt.Name]
		defaultVal := opt.Default

		var promptText string
		if opt.Required {
			promptText = fmt.Sprintf("%s%s", opt.Description, i18n.Msg(" (required)"))
		} else {
			promptText = opt.Description
		}

		switch opt.Type {
		case optionTypeString:
			defaultStr := ""
			if hasCurrent {
				defaultStr = fmt.Sprintf("%v", currentVal)
			} else if defaultVal != nil {
				defaultStr = fmt.Sprintf("%v", defaultVal)
			}

			val, _ := pterm.DefaultInteractiveTextInput.
				WithDefaultValue(defaultStr).
				Show(promptText)
			// Для обязательных опций не разрешаем пустое значение
			if opt.Required && val == "" {
				// Повторяем запрос, пока не получим непустое значение
				for val == "" {
					pterm.Warning.Println(i18n.Msg("This field is required"))
					val, _ = pterm.DefaultInteractiveTextInput.
						WithDefaultValue(defaultStr).
						Show(promptText)
				}
			}
			switch {
			case val != "":
				options[opt.Name] = val
			case opt.Default != nil:
				options[opt.Name] = opt.Default
			case opt.Required:
				// Если опция обязательна, но значение пустое и нет Default, всё равно добавляем
				options[opt.Name] = val
			}
		case optionTypeInt:
			defaultInt := 0
			if hasCurrent {
				if v, ok := currentVal.(int); ok {
					defaultInt = v
				}
			} else if defaultVal != nil {
				if v, ok := defaultVal.(int); ok {
					defaultInt = v
				}
			}

			val, _ := pterm.DefaultInteractiveTextInput.
				WithDefaultValue(fmt.Sprintf("%d", defaultInt)).
				Show(promptText)
			var intVal int
			_, _ = fmt.Sscanf(val, "%d", &intVal)
			options[opt.Name] = intVal
		case optionTypeBool:
			defaultBool := false
			if hasCurrent {
				if v, ok := currentVal.(bool); ok {
					defaultBool = v
				}
			} else if defaultVal != nil {
				if v, ok := defaultVal.(bool); ok {
					defaultBool = v
				}
			}

			val, _ := pterm.DefaultInteractiveConfirm.
				WithDefaultValue(defaultBool).
				Show(promptText)
			options[opt.Name] = val
		}
	}

	return
}

// promptPluginInitOptions запрашивает опции для команды plugin init с особой логикой:
// сначала спрашивает kind (выбор из списка), затем command только если kind == "command"
func promptPluginInitOptions(commandOptions []Option, currentOptions map[string]any, commandPath []string) (options map[string]any) {

	options = make(map[string]any)

	// Копируем текущие опции
	for k, v := range currentOptions {
		options[k] = v
	}

	// 1. Сначала спрашиваем kind (выбор из списка)
	kindOptions := []string{"pre", "stage", "command", "post"}
	currentKind, hasKind := currentOptions["kind"].(string)
	defaultIndex := 0
	if hasKind {
		for i, k := range kindOptions {
			if k == currentKind {
				defaultIndex = i
				break
			}
		}
	}

	selectedKind, _ := pterm.DefaultInteractiveSelect.
		WithOptions(kindOptions).
		WithDefaultOption(kindOptions[defaultIndex]).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(kindOptions))).
		Show(i18n.Msg("Plugin kind (required)"))

	if selectedKind == "" {
		// Если пользователь отменил, возвращаем nil
		return nil
	}

	options["kind"] = selectedKind

	// 2. Спрашиваем deploy-type (выбор из списка), если не указан
	currentDeployType, hasDeployType := currentOptions["deploy-type"].(string)
	if !hasDeployType || currentDeployType == "" {
		deployTypeOptions := []string{"none", "gitlab", "github"}
		deployTypeDefaultIndex := 0 // по умолчанию "none"

		selectedDeployType, _ := pterm.DefaultInteractiveSelect.
			WithOptions(deployTypeOptions).
			WithDefaultOption(deployTypeOptions[deployTypeDefaultIndex]).
			WithMaxHeight(utils.GetMaxHeightForSelect(len(deployTypeOptions))).
			Show(i18n.Msg("Deploy type"))

		if selectedDeployType == "" {
			// Если пользователь отменил, возвращаем nil
			return nil
		}

		options["deploy-type"] = selectedDeployType
	} else {
		// Если deploy-type уже указан, используем его
		options["deploy-type"] = currentDeployType
	}

	// 3. Если kind == "command", спрашиваем command
	if selectedKind == "command" {
		currentCommand, hasCommand := currentOptions["command"].(string)
		defaultCommand := ""
		if hasCommand {
			defaultCommand = currentCommand
		}

		commandVal, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultValue(defaultCommand).
			Show(i18n.Msg("CLI command name (required)"))

		// Для command требуем непустое значение
		for commandVal == "" {
			pterm.Warning.Println(i18n.Msg("This field is required"))
			commandVal, _ = pterm.DefaultInteractiveTextInput.
				WithDefaultValue(defaultCommand).
				Show(i18n.Msg("CLI command name (required)"))
		}

		if commandVal != "" {
			options["command"] = commandVal
		}
	}

	// 4. Запрашиваем остальные опции в обычном порядке
	for _, opt := range commandOptions {
		// Пропускаем kind, command и deploy-type, так как они уже обработаны
		if opt.Name == "kind" || opt.Name == "command" || opt.Name == "deploy-type" {
			continue
		}

		currentVal, hasCurrent := currentOptions[opt.Name]
		defaultVal := opt.Default

		var promptText string
		if opt.Required {
			promptText = fmt.Sprintf("%s%s", opt.Description, i18n.Msg(" (required)"))
		} else {
			promptText = opt.Description
		}

		switch opt.Type {
		case optionTypeString:
			defaultStr := ""
			if hasCurrent {
				defaultStr = fmt.Sprintf("%v", currentVal)
			} else if defaultVal != nil {
				defaultStr = fmt.Sprintf("%v", defaultVal)
			}

			val, _ := pterm.DefaultInteractiveTextInput.
				WithDefaultValue(defaultStr).
				Show(promptText)

			if opt.Required && val == "" {
				for val == "" {
					pterm.Warning.Println(i18n.Msg("This field is required"))
					val, _ = pterm.DefaultInteractiveTextInput.
						WithDefaultValue(defaultStr).
						Show(promptText)
				}
			}

			switch {
			case val != "":
				options[opt.Name] = val
			case opt.Default != nil:
				options[opt.Name] = opt.Default
			case opt.Required:
				options[opt.Name] = val
			}
		case optionTypeInt:
			defaultInt := 0
			if hasCurrent {
				if v, ok := currentVal.(int); ok {
					defaultInt = v
				}
			} else if defaultVal != nil {
				if v, ok := defaultVal.(int); ok {
					defaultInt = v
				}
			}

			val, _ := pterm.DefaultInteractiveTextInput.
				WithDefaultValue(fmt.Sprintf("%d", defaultInt)).
				Show(promptText)
			var intVal int
			_, _ = fmt.Sscanf(val, "%d", &intVal)
			options[opt.Name] = intVal
		case optionTypeBool:
			defaultBool := false
			if hasCurrent {
				if v, ok := currentVal.(bool); ok {
					defaultBool = v
				}
			} else if defaultVal != nil {
				if v, ok := defaultVal.(bool); ok {
					defaultBool = v
				}
			}

			val, _ := pterm.DefaultInteractiveConfirm.
				WithDefaultValue(defaultBool).
				Show(promptText)
			options[opt.Name] = val
		}
	}

	return
}

// PromptCommandOptionsFromPlugin запрашивает опции плагина интерактивно (для update команды)
func PromptCommandOptionsFromPlugin(commandOptions []Option, currentOptions map[string]any, commandPath []string) (options map[string]any) {

	// Если опций нет, возвращаем текущие опции
	if len(commandOptions) == 0 {
		options = currentOptions
		return
	}

	// Специальная логика для команды plugin init
	if len(commandPath) == 2 && commandPath[0] == "plugin" && commandPath[1] == "init" {
		return promptPluginInitOptions(commandOptions, currentOptions, commandPath)
	}

	return promptOptions(commandOptions, currentOptions)
}
