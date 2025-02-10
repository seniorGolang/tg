package jsonrpc

import (
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
}

type Option func(ops *options)

func prepareOpts(opts []Option) (options options) {

	options.customHeaders = make(map[string]string)
	for _, op := range opts {
		op(&options)
	}
	return
}

func HeaderFromCtx(headers ...interface{}) Option {
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
