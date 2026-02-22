// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"context"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"

	"github.com/goccy/go-json"
)

const (
	manifestJSONFilename = "manifest.json"
)

// DownloadManifest скачивает manifest.json из релиза по тегу и возвращает список путей к манифестам плагинов.
func (c *Client) DownloadManifest(ctx context.Context, tag string) (manifestPaths []string, err error) {

	manifestURL := fmt.Sprintf("%s/%s", fmt.Sprintf(GitHubReleasesBaseURL, c.owner, c.repo, tag), manifestJSONFilename)

	var body []byte
	if body, err = c.DownloadManifestContent(ctx, manifestURL); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to download manifest.json: %w"), err)
	}

	var manifests []string
	if err = json.Unmarshal(body, &manifests); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to parse manifest.json: %w"), err)
	}

	return manifests, nil
}
