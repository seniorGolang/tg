package httpclient

import (
	"crypto/tls"
	"time"

	"github.com/valyala/fasthttp"
)

type Option func(*ClientHTTP)

func WithClient(client *fasthttp.Client) Option {
	return func(c *ClientHTTP) {
		c.client = client
	}
}

func WithTLS(config *tls.Config) Option {
	return func(c *ClientHTTP) {
		c.client.TLSConfig = config
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(c *ClientHTTP) {
		c.client.ReadTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *ClientHTTP) {
		c.client.WriteTimeout = timeout
	}
}

func LogRequest() Option {
	return func(c *ClientHTTP) {
		c.logRequests = true
	}
}

func LogOnError() Option {
	return func(c *ClientHTTP) {
		c.logOnError = true
	}
}

func WithHeader(headers ...interface{}) Option {
	return func(c *ClientHTTP) {
		c.headersFromCtx = headers
	}
}
