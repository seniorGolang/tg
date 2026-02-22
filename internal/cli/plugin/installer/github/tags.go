// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	gitInfoRefsURLTemplate = "https://github.com/%s/%s.git/info/refs"
	gitServiceParam        = "?service=git-upload-pack"
	refsTagsPrefix         = "refs/tags/"
	tagSuffix              = "^{}"
	commentPrefix          = "#"
	newlineChar            = "\n"
	nullChar               = "\x00"
)

// listTags: git протокол, без GitHub API.
func (c *Client) listTags(ctx context.Context) (tags []string, err error) {

	gitURL := fmt.Sprintf(gitInfoRefsURLTemplate, c.owner, c.repo)

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, gitURL+gitServiceParam, nil); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
	}

	req.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	req.Header.Set("Accept", "*/*")

	var resp *http.Response
	//nolint:gosec // G704: URL репозитория GitHub из конфигурации
	if resp, err = c.httpClient.Do(req); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to execute request: %w"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body []byte
		body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf(i18n.Msg("Unexpected status code: %d, body: %s"), resp.StatusCode, string(body))
	}

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to read response: %w"), err)
	}

	tags = []string{}
	lines := strings.Split(string(body), newlineChar)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, commentPrefix) {
			continue
		}

		if !strings.Contains(line, refsTagsPrefix) {
			continue
		}

		parts := strings.Fields(line)
		for _, part := range parts {
			if strings.HasPrefix(part, refsTagsPrefix) {
				tag := strings.TrimPrefix(part, refsTagsPrefix)
				tag = strings.TrimSuffix(tag, tagSuffix)
				tag = strings.TrimRight(tag, nullChar)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
	}

	slog.Debug(i18n.Msg("Tags list received from repository"), "count", len(tags), "url", gitURL)

	return
}
