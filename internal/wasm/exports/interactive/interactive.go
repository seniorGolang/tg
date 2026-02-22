// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package interactive

import (
	"context"
	"errors"

	"github.com/seniorGolang/tg/v3/internal/cli/utils"
	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"

	"github.com/goccy/go-json"

	"github.com/pterm/pterm"
)

// Выполняет интерактивный выбор на стороне хоста через pterm.
// promptPtr, promptLen - указатель и размер строки с заголовком
// optionsPtr, optionsLen - указатель и размер JSON массива строк с опциями
// configPtr, configLen - указатель и размер JSON объекта с настройками (SelectConfig)
// resultPtrPtr, resultSizePtr - указатели на указатель и размер результата
//   - для одиночного выбора: JSON массив с одним элементом
//   - для множественного выбора: JSON массив строк с выбранными опциями
//
// Возвращает: 0 - успех, иначе код ошибки
func HostInteractiveSelect(ctx context.Context, h *host.Host, promptPtr uint32, promptLen uint32, optionsPtr uint32, optionsLen uint32, configPtr uint32, configLen uint32, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	if h.Malloc == nil {
		return 1
	}

	var err error
	var prompt string
	var options []string
	var config selectConfig
	if prompt, options, config, err = readSelectInputs(h, promptPtr, promptLen, optionsPtr, optionsLen, configPtr, configLen); err != nil || len(options) == 0 {
		return 1
	}

	var resultBytes []byte
	if resultBytes, err = performSelection(prompt, options, config); err != nil {
		return 1
	}

	return writeSelectResult(ctx, h, resultBytes, resultPtrPtr, resultSizePtr)
}

func readSelectInputs(h *host.Host, promptPtr uint32, promptLen uint32, optionsPtr uint32, optionsLen uint32, configPtr uint32, configLen uint32) (prompt string, options []string, config selectConfig, err error) {

	if prompt, err = memory.ReadString(h, promptPtr, promptLen); err != nil {
		return "", nil, selectConfig{}, err
	}

	if optionsLen > 0 {
		optionsJSONBytes, readErr := memory.ReadString(h, optionsPtr, optionsLen)
		if readErr != nil {
			return "", nil, selectConfig{}, readErr
		}
		if optionsJSONBytes != "" {
			if err = json.Unmarshal([]byte(optionsJSONBytes), &options); err != nil {
				return "", nil, selectConfig{}, err
			}
		}
	}

	if configLen > 0 {
		configJSONBytes, readErr := memory.ReadString(h, configPtr, configLen)
		if readErr != nil {
			return "", nil, selectConfig{}, readErr
		}
		if configJSONBytes != "" {
			var configJSON selectConfigJSON
			if err = json.Unmarshal([]byte(configJSONBytes), &configJSON); err != nil {
				config = selectConfig{MultiSelect: false, DefaultOptions: nil}
			} else {
				config.MultiSelect = configJSON.MultiSelect
				config.DefaultOptions = configJSON.DefaultOptions
			}
		}
	}

	return
}

// selectConfig содержит настройки для интерактивного выбора.
type selectConfig struct {
	MultiSelect    bool
	DefaultOptions []string
}

// selectConfigJSON представляет JSON формат конфигурации для десериализации.
type selectConfigJSON struct {
	MultiSelect    bool     `json:"multiSelect,omitempty"`
	DefaultOptions []string `json:"defaultOptions,omitempty"`
}

// selectResponse представляет JSON формат ответа интерактивного выбора.
type selectResponse struct {
	Selected []string `json:"selected"`
}

// performSelection выполняет интерактивный выбор в зависимости от настроек.
func performSelection(prompt string, options []string, config selectConfig) (result []byte, err error) {

	if config.MultiSelect {
		return performMultiSelect(prompt, options, config.DefaultOptions)
	}
	return performSingleSelect(prompt, options)
}

// performSingleSelect выполняет одиночный выбор.
func performSingleSelect(prompt string, options []string) (result []byte, err error) {

	var selected string
	if selected, err = pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options))).
		Show(prompt); err != nil || selected == "" {
		return nil, errors.New(i18n.Msg("selection cancelled or failed"))
	}

	respJSON := selectResponse{
		Selected: []string{selected},
	}
	return json.Marshal(respJSON)
}

// performMultiSelect выполняет множественный выбор.
func performMultiSelect(prompt string, options []string, defaultOptions []string) (result []byte, err error) {

	multiselect := pterm.DefaultInteractiveMultiselect.
		WithOptions(options).
		WithMaxHeight(utils.GetMaxHeightForSelect(len(options)))

	if len(defaultOptions) > 0 {
		multiselect = multiselect.WithDefaultOptions(defaultOptions)
	}

	var selected []string
	if selected, err = multiselect.Show(prompt); err != nil {
		return
	}

	respJSON := selectResponse{
		Selected: selected,
	}
	return json.Marshal(respJSON)
}

func writeSelectResult(ctx context.Context, h *host.Host, resultBytes []byte, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	if h == nil {
		return 0
	}
	return memory.WriteBytesToPtrSize(ctx, h, resultBytes, resultPtrPtr, resultSizePtr)
}
