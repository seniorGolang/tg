package jsonrpc

import (
	"context"
	"crypto/tls"
	"net/http"
)

type options struct {
	logOnError         bool
	logRequests        bool
	allowUnknownFields bool
	tlsConfig          *tls.Config
	clientHTTP         *http.Client
	headersFromCtx     []interface{}
	customHeaders      map[string]string
	before             func(ctx context.Context, req *http.Request) context.Context
	after              func(ctx context.Context, res *http.Response) error
}

type Option func(ops *options)

func prepareOpts(opts []Option) (options options) {

	options.customHeaders = make(map[string]string)
	for _, op := range opts {
		op(&options)
	}
	return
}

func BeforeRequest(before func(ctx context.Context, req *http.Request) context.Context) Option {
	return func(ops *options) {
		ops.before = before
	}
}

func AfterRequest(after func(ctx context.Context, res *http.Response) error) Option {
	return func(ops *options) {
		ops.after = after
	}
}

func HeaderFromCtx(headers ...any) Option {
	return func(ops *options) {
		ops.headersFromCtx = append(ops.headersFromCtx, headers...)
	}
}

func AllowUnknownFields(allowUnknownFields bool) Option {
	return func(ops *options) {
		ops.allowUnknownFields = allowUnknownFields
	}
}

func ClientHTTP(client *http.Client) Option {
	return func(ops *options) {
		ops.clientHTTP = client
	}
}

func ConfigTLS(tlsConfig *tls.Config) Option {
	return func(ops *options) {
		ops.tlsConfig = tlsConfig
	}
}

func LogRequest() Option {
	return func(ops *options) {
		ops.logRequests = true
	}
}

func LogOnError() Option {
	return func(ops *options) {
		ops.logOnError = true
	}
}
