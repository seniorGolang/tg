// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"fmt"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// PromptCommandArgs запрашивает позиционные аргументы для команды интерактивно
func PromptCommandArgs(cmd Command, cobraCmd *cobra.Command, commandPath []string) (args []string) {

	// Получаем позиционные опции из команды
	positionalOptions := getPositionalOptions(cmd.GetOptions())
	if len(positionalOptions) == 0 {
		args = nil
		return
	}

	// Запрашиваем аргументы по порядку позиционных опций
	args = make([]string, 0, len(positionalOptions))

	i := 0
	for i < len(positionalOptions) {
		opt := positionalOptions[i]

		// Формируем понятное сообщение на основе описания опции
		promptText := opt.Description
		if opt.Required {
			promptText = fmt.Sprintf("%s%s", promptText, i18n.Msg(" (required)"))
		}

		arg, _ := pterm.DefaultInteractiveTextInput.
			Show(promptText)

		if arg == "" {
			if opt.Required {
				// Для обязательных аргументов повторяем запрос
				pterm.Warning.Println(i18n.Msg("This field is required"))
				// Не увеличиваем i, повторяем для этой опции
				continue
			}
			// Для опциональных аргументов можно пропустить
			break
		}

		args = append(args, arg)

		// Проверяем валидность аргументов после каждого ввода
		if cobraCmd.Args != nil {
			if err := cobraCmd.Args(cobraCmd, args); err != nil {
				// Если аргументов недостаточно, продолжаем запрашивать
				if strings.Contains(err.Error(), "requires") || strings.Contains(err.Error(), "accepts") {
					// Нужно больше аргументов, переходим к следующей опции
					i++
					continue
				}
				// Другая ошибка - возможно, неверный формат
				pterm.Warning.Printfln(i18n.Msg("Invalid argument: %s"), err)
				// Удаляем последний аргумент и пробуем снова
				args = args[:len(args)-1]
				// Не увеличиваем i, повторяем для этой опции
				continue
			}
		}

		// Если валидация прошла успешно и это был последний обязательный аргумент, выходим
		// Проверяем, есть ли ещё обязательные аргументы после текущего
		hasMoreRequired := false
		for j := i + 1; j < len(positionalOptions); j++ {
			if positionalOptions[j].Required {
				hasMoreRequired = true
				break
			}
		}

		// Если больше нет обязательных аргументов и валидация прошла, возвращаем результат
		if !hasMoreRequired {
			return
		}

		// Переходим к следующей опции
		i++
	}

	// Валидация прошла успешно
	return
}
