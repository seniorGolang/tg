package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type ClientHTTP struct {
	client         *fasthttp.Client
	BaseURL        string
	logRequests    bool
	logOnError     bool
	headersFromCtx []interface{}
}

// NewClient creates a new HTTP client with the provided baseURL, circuit breaker settings, and options.
func NewClient(baseURL string, opts ...Option) *ClientHTTP {

	c := &ClientHTTP{
		client: &fasthttp.Client{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		BaseURL: baseURL,
	}
	// Apply options
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do send the request and fills the response, using the circuit breaker.
func (c *ClientHTTP) Do(ctx context.Context, req *fasthttp.Request, resp *fasthttp.Response) (err error) {

	// Set headers from context
	for _, header := range c.headersFromCtx {
		if value := ctx.Value(header); value != nil {
			if k := toString(header); k != "" {
				if v := toString(value); v != "" {
					req.Header.Set(k, v)
				}
			}
		}
	}
	// Log the request if logRequests is enabled
	if c.logRequests {
		if cmd, cmdErr := toCurlCommand(req); cmdErr == nil {
			log.Debug().Str("method", string(req.Header.Method())).
				Str("url", req.URI().String()).
				Str("curl", cmd.String()).
				Msg("HTTP request")
		}
	}
	// Defer function to log on error if logOnError is enabled
	defer func() {
		if err != nil && c.logOnError {
			if cmd, cmdErr := toCurlCommand(req); cmdErr == nil {
				log.Error().Str("method", string(req.Header.Method())).
					Str("url", req.URI().String()).
					Str("curl", cmd.String()).
					Err(err).
					Msg("HTTP request failed")
			}
		}
	}()
	err = c.client.Do(req, resp)
	return
}

// SetTimeout sets the read and write timeout for the client.
func (c *ClientHTTP) SetTimeout(timeout time.Duration) {
	c.client.ReadTimeout = timeout
	c.client.WriteTimeout = timeout
}

// Option represents a configuration option for ClientHTTP.
type Option func(*ClientHTTP)

// WithTimeout sets the read and write timeout for the client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *ClientHTTP) {
		c.SetTimeout(timeout)
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

func HeaderFromCtx(headers ...interface{}) Option {
	return func(c *ClientHTTP) {
		c.headersFromCtx = headers
	}
}

func toCurlCommand(req *fasthttp.Request) (*bytes.Buffer, error) {

	var cmd bytes.Buffer
	cmd.WriteString("curl -X ")
	cmd.WriteString(string(req.Header.Method())) // nolint:mirror
	cmd.WriteString(" '")
	cmd.WriteString(req.URI().String())
	cmd.WriteString("'")
	// Add headers
	req.Header.VisitAll(func(key, value []byte) {
		cmd.WriteString(" -H '")
		cmd.WriteString(string(key)) // nolint:mirror
		cmd.WriteString(": ")
		cmd.WriteString(string(value)) // nolint:mirror
		cmd.WriteString("'")
	})
	// Add body if present
	if req.Body() != nil && len(req.Body()) > 0 {
		cmd.WriteString(" -d '")
		cmd.WriteString(string(req.Body())) // nolint:mirror
		cmd.WriteString("'")
	}
	return &cmd, nil
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return ""
	}
}
