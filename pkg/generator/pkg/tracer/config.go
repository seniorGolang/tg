package tracer

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type config struct {
	Port                   *int
	ServerName             *string
	collectClientIP        bool
	Next                   func(*fiber.Ctx) bool
	Propagators            propagation.TextMapPropagator
	MeterProvider          otelmetric.MeterProvider
	TracerProvider         oteltrace.TracerProvider
	CustomAttributes       func(*fiber.Ctx) []attribute.KeyValue
	SpanNameFormatter      func(*fiber.Ctx) string
	CustomMetricAttributes func(*fiber.Ctx) []attribute.KeyValue
}

type Option interface {
	apply(cfg *config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithNext(f func(ctx *fiber.Ctx) bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.Next = f
	})
}

func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		cfg.Propagators = propagators
	})
}

func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.MeterProvider = provider
	})
}

func WithSpanNameFormatter(f func(ctx *fiber.Ctx) string) Option {
	return optionFunc(func(cfg *config) {
		cfg.SpanNameFormatter = f
	})
}

func WithServerName(serverName string) Option {
	return optionFunc(func(cfg *config) {
		cfg.ServerName = &serverName
	})
}

func WithPort(port int) Option {
	return optionFunc(func(cfg *config) {
		cfg.Port = &port
	})
}

func WithCustomAttributes(f func(ctx *fiber.Ctx) []attribute.KeyValue) Option {
	return optionFunc(func(cfg *config) {
		cfg.CustomAttributes = f
	})
}

func WithCustomMetricAttributes(f func(ctx *fiber.Ctx) []attribute.KeyValue) Option {
	return optionFunc(func(cfg *config) {
		cfg.CustomMetricAttributes = f
	})
}

func WithCollectClientIP(collect bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.collectClientIP = collect
	})
}
