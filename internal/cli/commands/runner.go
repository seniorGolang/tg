// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/state"
	"github.com/seniorGolang/tg/v3/internal/wasm/imports"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func createCommandRunner(cmd Command, rootDir string) (runner func(cobraCmd *cobra.Command, args []string)) {
	return func(cobraCmd *cobra.Command, args []string) {
		// 0. Проверяем флаг --version на корневой команде (до всех остальных действий)
		// Проверяем, был ли флаг установлен явно пользователем
		rootCmd := cobraCmd.Root()
		if rootCmd.PersistentFlags().Changed("version") {
			// Проверяем, является ли команда плагином
			if pluginCmd, ok := cmd.(*lazyPluginCommand); ok {
				var pluginVersion string
				var err error
				if pluginVersion, err = pluginCmd.GetPluginVersion(); err != nil {
					slog.Error(i18n.Msg("Failed to get plugin version"), "error", err)
					return
				}
				pluginName := pluginCmd.metadata.pluginName
				versionText := i18n.Msg("Version")
				pterm.Print(pterm.Green(versionText), " ", pterm.Cyan(pluginName), " ", pluginVersion, "\n")
				return
			}
			// Для встроенных команд версия не поддерживается
			slog.Error(i18n.Msg("Version information is not available for builtin commands"))
			return
		}

		// 1. Извлекаем глобальные опции из корневой команды
		globalOpts := extractGlobalOptions(cobraCmd)

		// 2. Настраиваем уровень логирования на основе глобальной опции
		logger := createLoggerWithLevel(globalOpts.LogLevel)

		// Обновляем глобальный slog для всех мест, которые используют slog напрямую
		// Это нужно, чтобы --log-level работал для всех логов, не только для ctx.Logger
		updateGlobalSlogLevel(globalOpts.LogLevel)

		// 3. Извлекаем опции команды из флагов (исключая позиционные)
		options := extractOptionsFromFlags(cobraCmd, cmd.GetOptions())

		// 4. Получаем путь команды
		commandPath := getCommandPath(cobraCmd)

		// 5. Проверяем, нужен ли интерактивный режим для опций
		// Если команда не имеет опций или все опции имеют значения → выполняем сразу
		// Исключаем позиционные опции из проверки
		nonPositionalOptions := getNonPositionalOptions(cmd.GetOptions())
		hasOptions := len(nonPositionalOptions) > 0
		hasRequiredNonPositionalOptions := hasRequiredOptions(nonPositionalOptions)
		allOptionsProvided := validateRequiredOptions(options, nonPositionalOptions)

		if hasOptions && hasRequiredNonPositionalOptions && !allOptionsProvided {
			// Обработка отсутствующих обязательных опций
			if globalOpts.FailOnMissing {
				// Выводим ошибку вместо интерактивного режима
				requiredOpts := getRequiredOptions(nonPositionalOptions)
				slog.Error(i18n.Msg("Required options are missing"),
					"command", strings.Join(commandPath, " "),
					"required_options", requiredOpts)
				return
			}

			// Запускаем интерактивный режим
			options = PromptCommandOptions(cmd, options, commandPath)
			if options == nil {
				return // Пользователь отменил
			}
		}
		// Если опций нет или все опции предоставлены → выполняем сразу

		// 6. Проверяем позиционные аргументы
		// Если команда требует позиционные аргументы, но они не переданы, запрашиваем интерактивно
		positionalOptions := getPositionalOptions(cmd.GetOptions())
		hasRequiredPositional := hasRequiredOptions(positionalOptions)

		// Проверяем, достаточно ли передано аргументов для обязательных позиционных опций
		needsInteractiveArgs := false
		if hasRequiredPositional {
			requiredCount := 0
			for _, opt := range positionalOptions {
				if opt.Required {
					requiredCount++
				}
			}
			if len(args) < requiredCount {
				needsInteractiveArgs = true
			}
		}

		// Если есть валидатор cobra, проверяем через него
		if cobraCmd.Args != nil {
			if err := cobraCmd.Args(cobraCmd, args); err != nil {
				needsInteractiveArgs = true
			}
		}

		if needsInteractiveArgs {
			// Если аргументы не соответствуют требованиям, запрашиваем интерактивно
			if globalOpts.FailOnMissing {
				slog.Error(i18n.Msg("Required positional arguments are missing"),
					"command", strings.Join(commandPath, " "))
				return
			}

			// Запрашиваем позиционные аргументы интерактивно
			args = PromptCommandArgs(cmd, cobraCmd, commandPath)
			if args == nil {
				return // Пользователь отменил
			}
		}

		// 7. Показываем собранную команду после всех интерактивных запросов (если не скрыто)
		if !globalOpts.HideCmd {
			showBuiltCommand(cmd, options, args, commandPath)
		}

		// 8. Получаем контекст из cobra команды (если установлен) или создаем новый
		cmdCtx := cobraCmd.Context()
		if cmdCtx == nil {
			cmdCtx = context.Background()
		}

		// 9. Создаём контекст выполнения
		ctx := CommandContext{
			Context:      cmdCtx,
			RootDir:      rootDir,
			Options:      options,
			Args:         args, // Позиционные аргументы
			CommandPath:  commandPath,
			Logger:       logger,
			StateManager: state.New(rootDir),
			GlobalOpts:   globalOpts,
		}

		// 10. Выполняем команду
		if err := cmd.Execute(ctx); err != nil {
			// Извлекаем оригинальное сообщение из ошибки плагина, если это ошибка плагина
			var pluginErr *imports.PluginError
			if errors.As(err, &pluginErr) {
				slog.Error(pluginErr.Message)
			} else {
				// Для других ошибок выводим полное сообщение
				slog.Error(i18n.Msg("Command execution error"), "error", err)
			}
		}
	}
}
