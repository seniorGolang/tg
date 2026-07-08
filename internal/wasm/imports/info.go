// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package imports

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/plugin"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// Info использует глобальный канал для вызова WASM функции info; request передаётся через указатель на память.
func Info(ctx context.Context, h *host.Host) (info plugin.Info, err error) {

	if ctx.Err() != nil {
		return plugin.Info{}, fmt.Errorf("%s: %w", i18n.Msg("context cancelled"), ctx.Err())
	}

	req := infoRequest{}

	var resp *infoResponse
	var requestData []byte
	if requestData, err = json.Marshal(req); err != nil {
		return plugin.Info{}, fmt.Errorf("%s: %w", i18n.Msg("failed to marshal request"), err)
	}

	if resp, err = callWithResult[infoResponse](ctx, h, h.CallChannel, wasm.FuncInfo, requestData); err != nil {
		return plugin.Info{}, fmt.Errorf("%s: %w", i18n.Msg("failed to call info"), err)
	}

	if resp != nil {
		info = resp.Info
	}

	return
}
