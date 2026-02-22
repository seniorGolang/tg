// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package github

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

const (
	httpDialTimeout           = 10 * time.Second
	httpClientTimeout         = 30 * time.Second
	httpResponseHeaderTimeout = 10 * time.Second
	httpTLSHandshakeTimeout   = 10 * time.Second
)

func NewClient(repoURL string) (client *Client, err error) {

	var parsedURL *url.URL
	if parsedURL, err = url.Parse(repoURL); err != nil {
		err = fmt.Errorf(i18n.Msg("Invalid repository URL: %w"), err)
		return
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) < 2 {
		err = errors.New(i18n.Msg("Invalid repository URL format, expected: https://github.com/owner/repo"))
		return
	}

	owner := parts[0]
	repo := parts[1]

	dialer := &net.Dialer{
		Timeout: httpDialTimeout,
	}

	client = &Client{
		owner: owner,
		repo:  repo,
		httpClient: &http.Client{
			Timeout: httpClientTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.DialContext(ctx, network, addr)
				},
				TLSHandshakeTimeout:   httpTLSHandshakeTimeout,
				ResponseHeaderTimeout: httpResponseHeaderTimeout,
			},
		},
	}
	return
}
