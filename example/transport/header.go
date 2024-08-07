// GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.
package transport

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	opentracinggo "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/seniorGolang/json"
)

type Header struct {
	SpanKey       string
	SpanValue     interface{}
	RequestKey    string
	RequestValue  interface{}
	ResponseKey   string
	ResponseValue interface{}
	LogKey        string
	LogValue      interface{}
}

type HeaderHandler func(value string) Header

func (srv *Server) headersHandler(ctx *fiber.Ctx) error {

	span := makeSpan(ctx, fmt.Sprintf("request:%s", ctx.Path()))
	defer injectSpan(ctx, span)
	defer span.Finish()
	for headerName, handler := range srv.headerHandlers {
		value := ctx.Request().Header.Peek(headerName)
		header := handler(string(value))
		if header.RequestValue != nil {
			ctx.Request().Header.Set(header.RequestKey, headerValue(header.RequestValue))
		}
		if header.ResponseValue != nil {
			ctx.Response().Header.Set(header.ResponseKey, headerValue(header.ResponseValue))
		}
		if header.LogValue != nil {
			logger := log.Ctx(ctx.UserContext()).With().Interface(header.LogKey, header.LogValue).Logger()
			ctx.SetUserContext(logger.WithContext(ctx.UserContext()))
		}
	}
	ctx.SetUserContext(opentracinggo.ContextWithSpan(ctx.UserContext(), span))
	return ctx.Next()
}

func headerValue(src interface{}) (value string) {

	switch src.(type) {
	case string:
		return src.(string)
	case iHeaderValue:
		return src.(iHeaderValue).Header()
	case fmt.Stringer:
		return src.(fmt.Stringer).String()
	default:
		bytes, _ := json.Marshal(src)
		return string(bytes)
	}
}

type iHeaderValue interface {
	Header() string
}
