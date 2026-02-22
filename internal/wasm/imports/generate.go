// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package imports

import (
	"context"
	"fmt"

	"github.com/goccy/go-json"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

func Generate(ctx context.Context, h *host.Host, rootDir string, moduleName string) (err error) {

	if ctx.Err() != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("context cancelled"), ctx.Err())
	}

	req := generateRequest{
		RootDir:    rootDir,
		ModuleName: moduleName,
	}

	var requestData []byte
	if requestData, err = json.Marshal(req); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("failed to marshal request"), err)
	}

	var resp *generateResponse
	if resp, err = callWithResult[generateResponse](ctx, h, h.CallChannel, wasm.FuncGenerate, requestData); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("failed to call generate"), err)
	}

	if resp != nil && resp.Error != "" {
		return NewPluginError(resp.Error)
	}

	return
}
