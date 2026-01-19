// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package imports

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// Execute выполняет плагин с переданными данными.
// Использует глобальный канал для вызова WASM функции execute.
// Все данные сериализуются в JSON и передаются через указатель на память.
func Execute(ctx context.Context, h *host.Host, rootDir string, request plugin.Storage, path ...string) (response plugin.Storage, err error) {

	startTime := time.Now()

	// Выводим данные запроса в момент запуска
	var requestJSON []byte
	if request != nil {
		var marshalErr error
		if requestJSON, marshalErr = json.Marshal(request); marshalErr != nil {
			slog.Debug(i18n.Msg("Execute: failed to marshal request for logging"),
				slog.String("rootDir", rootDir),
				slog.Any("path", path),
				slog.String("error", marshalErr.Error()),
			)
		} else {
			slog.Debug(i18n.Msg("Execute: starting"),
				slog.String("rootDir", rootDir),
				slog.Any("path", path),
				slog.String("request", string(requestJSON)),
			)
		}
	} else {
		slog.Debug(i18n.Msg("Execute: starting"),
			slog.String("rootDir", rootDir),
			slog.Any("path", path),
			slog.String("request", "null"),
		)
	}

	// Выводим данные ответа и время работы в момент завершения
	defer func() {
		duration := time.Since(startTime)
		var responseJSON []byte
		var marshalErr error
		if response != nil {
			if responseJSON, marshalErr = json.Marshal(response); marshalErr != nil {
				slog.Debug(i18n.Msg("Execute: completed"),
					slog.Duration("duration", duration),
					slog.String("error", i18n.Msg("failed to marshal response for logging")),
					slog.Any("marshalError", marshalErr),
				)
			} else {
				if err != nil {
					slog.Debug(i18n.Msg("Execute: completed with error"),
						slog.Duration("duration", duration),
						slog.String("error", err.Error()),
						slog.String("response", string(responseJSON)),
					)
				} else {
					slog.Debug(i18n.Msg("Execute: completed successfully"),
						slog.Duration("duration", duration),
						slog.String("response", string(responseJSON)),
					)
				}
			}
		} else {
			if err != nil {
				slog.Debug(i18n.Msg("Execute: completed with error"),
					slog.Duration("duration", duration),
					slog.String("error", err.Error()),
					slog.String("response", "null"),
				)
			} else {
				slog.Debug(i18n.Msg("Execute: completed successfully"),
					slog.Duration("duration", duration),
					slog.String("response", "null"),
				)
			}
		}
	}()

	if ctx.Err() != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("context cancelled"), ctx.Err())
	}

	// Подготавливаем запрос
	req := executeRequest{
		RootDir: rootDir,
		Request: request,
		Path:    path,
	}

	// Сериализуем запрос в JSON
	var requestData []byte
	if requestData, err = json.Marshal(req); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to marshal request"), err)
	}

	// Вызываем функцию через глобальный канал
	var resp *executeResponse
	if resp, err = callWithResult[executeResponse](ctx, h, h.CallChannel, "execute", requestData); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to call execute"), err)
	}

	if resp != nil {
		if resp.Error != "" {
			return nil, NewPluginError(resp.Error)
		}

		if len(resp.Response) > 0 {
			mapStorage := resp.Response
			response = &mapStorage
		}
	}

	if response == nil {
		response = plugin.NewStorage()
	}

	// ВАЖНО: Ждем завершения всех активных HTTP серверов, listener'ов и задач
	// Execute не должен завершаться, пока есть активные серверы, listener'ы или задачи
	// Это гарантирует, что серверы и задачи будут работать, пока плагин выполняется
	// Серверы будут остановлены при закрытии listener'ов или отмене контекста
	serversChan := make(chan struct{})
	listenersChan := make(chan struct{})
	tasksChan := make(chan struct{})

	go func() {
		h.ActiveServers.Wait()
		close(serversChan)
	}()

	go func() {
		h.ActiveListeners.Wait()
		close(listenersChan)
	}()

	go func() {
		h.ActiveTasks.Wait()
		close(tasksChan)
	}()

	// Ждем, пока все активные ресурсы не завершатся или контекст не будет отменен
	// Используем отдельные горутины для каждого WaitGroup, чтобы не блокировать друг друга
	// select завершится, когда все каналы закроются (все WaitGroup завершены) или контекст отменен
	done := make(chan struct{})
	go func() {
		// Ждем, пока все три WaitGroup не завершатся
		<-serversChan
		<-listenersChan
		<-tasksChan
		close(done)
	}()

	select {
	case <-done:
		// Все серверы, listener'ы и задачи завершены
	case <-ctx.Done():
		// Контекст отменен, серверы и задачи будут остановлены при закрытии listener'ов
	}

	return
}
