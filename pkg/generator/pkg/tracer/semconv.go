package tracer

import (
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func httpServerMetricAttributesFromRequest(c *fiber.Ctx, cfg config) []attribute.KeyValue {

	attrs := []attribute.KeyValue{
		httpFlavorAttribute(c),
		semconv.HTTPMethodKey.String(utils.CopyString(c.Method())),
		semconv.HTTPSchemeKey.String(utils.CopyString(c.Protocol())),
		semconv.NetHostNameKey.String(utils.CopyString(c.Hostname())),
	}
	if cfg.Port != nil {
		attrs = append(attrs, semconv.NetHostPortKey.Int(*cfg.Port))
	}
	if cfg.ServerName != nil {
		attrs = append(attrs, semconv.HTTPServerNameKey.String(*cfg.ServerName))
	}
	if cfg.CustomMetricAttributes != nil {
		attrs = append(attrs, cfg.CustomMetricAttributes(c)...)
	}
	return attrs
}

func httpServerTraceAttributesFromRequest(c *fiber.Ctx, cfg config) []attribute.KeyValue {

	attrs := []attribute.KeyValue{
		httpFlavorAttribute(c),
		semconv.HTTPMethodKey.String(utils.CopyString(c.Method())),
		semconv.HTTPRequestContentLengthKey.Int(c.Request().Header.ContentLength()),
		semconv.HTTPSchemeKey.String(utils.CopyString(c.Protocol())),
		semconv.HTTPTargetKey.String(string(utils.CopyBytes(c.Request().RequestURI()))),
		semconv.HTTPURLKey.String(utils.CopyString(c.OriginalURL())),
		semconv.HTTPUserAgentKey.String(string(utils.CopyBytes(c.Request().Header.UserAgent()))),
		semconv.NetHostNameKey.String(utils.CopyString(c.Hostname())),
		semconv.NetTransportTCP,
	}
	if cfg.Port != nil {
		attrs = append(attrs, semconv.NetHostPortKey.Int(*cfg.Port))
	}
	if cfg.ServerName != nil {
		attrs = append(attrs, semconv.HTTPServerNameKey.String(*cfg.ServerName))
	}
	if username, ok := HasBasicAuth(c.Get(fiber.HeaderAuthorization)); ok {
		attrs = append(attrs, semconv.EnduserIDKey.String(utils.CopyString(username)))
	}
	if cfg.collectClientIP {
		clientIP := c.IP()
		if len(clientIP) > 0 {
			attrs = append(attrs, semconv.HTTPClientIPKey.String(utils.CopyString(clientIP)))
		}
	}
	if cfg.CustomAttributes != nil {
		attrs = append(attrs, cfg.CustomAttributes(c)...)
	}
	return attrs
}

func httpFlavorAttribute(c *fiber.Ctx) attribute.KeyValue {

	if c.Request().Header.IsHTTP11() {
		return semconv.HTTPFlavorHTTP11
	}
	return semconv.HTTPFlavorHTTP10
}

func HasBasicAuth(auth string) (string, bool) {

	if auth == "" {
		return "", false
	}
	if !strings.HasPrefix(auth, "Basic ") {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return "", false
	}
	creds := utils.UnsafeString(raw)
	index := strings.Index(creds, ":")
	if index == -1 {
		return "", false
	}
	return creds[:index], true
}
