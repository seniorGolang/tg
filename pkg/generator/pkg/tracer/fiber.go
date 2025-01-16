package tracer

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerKey           = "tracer-tg"
	instrumentationName = "tg"

	MetricNameHttpServerDuration       = "http.server.duration"
	MetricNameHttpServerRequestSize    = "http.server.request.size"
	MetricNameHttpServerResponseSize   = "http.server.response.size"
	MetricNameHttpServerActiveRequests = "http.server.active_requests"

	UnitDimensionless = "1"
	UnitBytes         = "By"
	UnitMilliseconds  = "ms"
)

func Middleware(opts ...Option) fiber.Handler {

	cfg := config{
		collectClientIP: true,
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(contrib.Version()),
	)
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(contrib.Version()),
	)
	httpServerDuration, err := meter.Float64Histogram(MetricNameHttpServerDuration, metric.WithUnit(UnitMilliseconds), metric.WithDescription("measures the duration inbound HTTP requests"))
	if err != nil {
		otel.Handle(err)
	}
	httpServerRequestSize, err := meter.Int64Histogram(MetricNameHttpServerRequestSize, metric.WithUnit(UnitBytes), metric.WithDescription("measures the size of HTTP request messages"))
	if err != nil {
		otel.Handle(err)
	}
	httpServerResponseSize, err := meter.Int64Histogram(MetricNameHttpServerResponseSize, metric.WithUnit(UnitBytes), metric.WithDescription("measures the size of HTTP response messages"))
	if err != nil {
		otel.Handle(err)
	}
	httpServerActiveRequests, err := meter.Int64UpDownCounter(MetricNameHttpServerActiveRequests, metric.WithUnit(UnitDimensionless), metric.WithDescription("measures the number of concurrent HTTP requests that are currently in-flight"))
	if err != nil {
		otel.Handle(err)
	}
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.SpanNameFormatter == nil {
		cfg.SpanNameFormatter = defaultSpanNameFormatter
	}
	return func(ftx *fiber.Ctx) error {

		if cfg.Next != nil && cfg.Next(ftx) {
			return ftx.Next()
		}
		ftx.Locals(tracerKey, tracer)
		savedCtx, cancel := context.WithCancel(ftx.UserContext())
		start := time.Now()
		requestMetricsAttrs := httpServerMetricAttributesFromRequest(ftx, cfg)
		httpServerActiveRequests.Add(savedCtx, 1, metric.WithAttributes(requestMetricsAttrs...))
		responseMetricAttrs := make([]attribute.KeyValue, 0, len(requestMetricsAttrs))
		copy(responseMetricAttrs, requestMetricsAttrs)
		reqHeader := make(http.Header)
		var reqHeaderAttrs []attribute.KeyValue
		ftx.Request().Header.VisitAll(func(k, v []byte) {
			reqHeader.Add(string(k), string(v))
			if strings.HasPrefix(strings.ToLower(string(k)), "x-") {
				reqHeaderAttrs = append(reqHeaderAttrs, attribute.String(fmt.Sprintf("header.%s", string(k)), string(v)))
			}
		})
		req := http.Request{Header: reqHeader}
		for _, cookie := range req.Cookies() {
			if strings.HasPrefix(strings.ToLower(cookie.Name), "x-") {
				reqHeaderAttrs = append(reqHeaderAttrs, attribute.String(fmt.Sprintf("cookie.%s", cookie.Name), cookie.Value))
			}
		}
		ctx := cfg.Propagators.Extract(savedCtx, propagation.HeaderCarrier(reqHeader))
		options := []trace.SpanStartOption{
			trace.WithAttributes(httpServerTraceAttributesFromRequest(ftx, cfg)...),
			trace.WithSpanKind(trace.SpanKindServer),
		}
		spanName := utils.CopyString(ftx.Path())
		ctx, span := tracer.Start(ctx, spanName, options...)
		defer span.End()
		ftx.SetUserContext(ctx)
		if err = ftx.Next(); err != nil {
			span.RecordError(err)
			_ = ftx.App().Config().ErrorHandler(ftx, err)
		}
		responseAttrs := append(
			semconv.HTTPAttributesFromHTTPStatusCode(ftx.Response().StatusCode()),
			append(reqHeaderAttrs, semconv.HTTPRouteKey.String(ftx.Route().Path))...,
		)
		var responseSize int64
		requestSize := int64(len(ftx.Request().Body()))
		if ftx.GetRespHeader("Content-Type") != "text/event-stream" {
			responseSize = int64(len(ftx.Response().Body()))
		}
		defer func() {
			responseMetricAttrs = append(responseMetricAttrs, responseAttrs...)
			httpServerActiveRequests.Add(savedCtx, -1, metric.WithAttributes(requestMetricsAttrs...))
			httpServerDuration.Record(savedCtx, float64(time.Since(start).Microseconds())/1000, metric.WithAttributes(responseMetricAttrs...))
			httpServerRequestSize.Record(savedCtx, requestSize, metric.WithAttributes(responseMetricAttrs...))
			httpServerResponseSize.Record(savedCtx, responseSize, metric.WithAttributes(responseMetricAttrs...))

			ftx.SetUserContext(savedCtx)
			cancel()
		}()
		span.SetAttributes(
			append(
				responseAttrs,
				semconv.HTTPResponseContentLengthKey.Int64(responseSize),
			)...)
		span.SetName(cfg.SpanNameFormatter(ftx))
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(ftx.Response().StatusCode(), trace.SpanKindServer)
		span.SetStatus(spanStatus, spanMessage)
		tracingHeaders := make(propagation.HeaderCarrier)
		cfg.Propagators.Inject(ftx.UserContext(), tracingHeaders)
		for _, headerKey := range tracingHeaders.Keys() {
			ftx.Set(headerKey, tracingHeaders.Get(headerKey))
		}
		return nil
	}
}

func defaultSpanNameFormatter(ctx *fiber.Ctx) string {
	return ctx.Route().Path
}
