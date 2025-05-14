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

func NewClient(baseURL string, opts ...Option) *ClientHTTP {

	c := &ClientHTTP{
		client: &fasthttp.Client{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		BaseURL: baseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ClientHTTP) Do(ctx context.Context, req *fasthttp.Request, resp *fasthttp.Response) (err error) {

	for _, header := range c.headersFromCtx {
		if value := ctx.Value(header); value != nil {
			if k := toString(header); k != "" {
				if v := toString(value); v != "" {
					req.Header.Set(k, v)
				}
			}
		}
	}
	if c.logRequests {
		if cmd, cmdErr := toCurlCommand(req); cmdErr == nil {
			log.Debug().Str("method", string(req.Header.Method())).
				Str("curl", cmd.String()).
				Msg("HTTP request")
		}
	}
	defer func() {
		if err != nil && c.logOnError {
			if cmd, cmdErr := toCurlCommand(req); cmdErr == nil {
				log.Error().Str("method", string(req.Header.Method())).
					Str("curl", cmd.String()).
					Err(err).
					Msg("HTTP request failed")
			}
		}
	}()
	err = c.client.Do(req, resp)
	return
}

func (c *ClientHTTP) SetTimeout(timeout time.Duration) {
	c.client.ReadTimeout = timeout
	c.client.WriteTimeout = timeout
}

func toCurlCommand(req *fasthttp.Request) (*bytes.Buffer, error) {

	var cmd bytes.Buffer
	cmd.WriteString("curl -X ")
	cmd.Write(req.Header.Method())
	cmd.WriteString(" '")
	cmd.WriteString(req.URI().String())
	cmd.WriteString("'")
	req.Header.VisitAll(func(key, value []byte) {
		cmd.WriteString(" -H '")
		cmd.Write(key)
		cmd.WriteString(": ")
		cmd.Write(value)
		cmd.WriteString("'")
	})
	if req.Body() != nil && len(req.Body()) > 0 {
		cmd.WriteString(" -d '")
		cmd.Write(req.Body())
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
		return fmt.Sprint(v)
	}
}
