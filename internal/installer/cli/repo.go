// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

// HandleRepo обрабатывает команду repo.
func (inst *Installer) HandleRepo(ctx context.Context, args []string, force bool) (err error) {

	if len(args) == 0 {
		return errors.New(i18n.Msg("Manifest URL not specified"))
	}

	url := args[0]
	if !strings.HasSuffix(url, manifestYAMLExt) && !strings.HasSuffix(url, manifestYMLExt) {
		url = strings.TrimSuffix(url, "/") + "/" + storage.ManifestFileName
	}

	source := storage.ExtractSourceFromManifestURL(url)
	if _, err = inst.manifestManager.LoadManifestCascade(ctx, url, source, force); err != nil {
		return
	}
	return
}
