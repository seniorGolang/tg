// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

type logRecord struct {
	level   slog.Level
	message string
	attrs   map[string]string
}

type captureHandler struct {
	records []logRecord
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) (enabled bool) {

	return true
}

func (h *captureHandler) Handle(_ context.Context, record slog.Record) (err error) {

	entry := logRecord{
		level:   record.Level,
		message: record.Message,
		attrs:   make(map[string]string),
	}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.String()
		return true
	})
	h.records = append(h.records, entry)
	return
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) (handler slog.Handler) {

	return h
}

func (h *captureHandler) WithGroup(_ string) (handler slog.Handler) {

	return h
}

func TestBuildFailureErrorWithoutOutput(t *testing.T) {

	cause := errors.New("exit status 1")
	failure := newBuildFailure("swagger", phaseCompile, cause, "")

	got := failure.Error()
	if !strings.Contains(got, "swagger") {
		t.Fatalf("expected plugin name in error, got %q", got)
	}
	if !strings.Contains(got, phaseCompile) {
		t.Fatalf("expected phase in error, got %q", got)
	}
	if strings.Contains(got, "unknown field") {
		t.Fatalf("output must not be embedded in error message, got %q", got)
	}
}

func TestBuildFailureUnwrap(t *testing.T) {

	cause := errors.New("exit status 1")
	failure := newBuildFailure("swagger", phaseCompile, cause, "compiler output")

	if !errors.Is(failure, cause) {
		t.Fatalf("expected unwrap to return cause")
	}
}

func TestLogBuildFailureIncludesOutput(t *testing.T) {

	handler := &captureHandler{}
	logger := slog.New(handler)
	ctx := context.Background()

	failure := newBuildFailure("swagger", phaseCompile, errors.New("exit status 1"), "compiler output")
	logger.ErrorContext(ctx, "Plugin build failed",
		slog.String("plugin", failure.Plugin),
		slog.String("phase", failure.Phase),
		slog.String("error", failure.Cause.Error()),
		slog.String("output", failure.Output),
	)

	if len(handler.records) != 1 {
		t.Fatalf("expected one log record, got %d", len(handler.records))
	}
	if handler.records[0].attrs["output"] != "compiler output" {
		t.Fatalf("expected output attr, got %#v", handler.records[0].attrs)
	}
}

func TestLogBuildFailureOmitsEmptyOutput(t *testing.T) {

	handler := &captureHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(prev)

	logBuildFailure(context.Background(), newBuildFailure("swagger", phaseMetadata, errors.New("load failed"), ""))

	if len(handler.records) != 1 {
		t.Fatalf("expected one log record, got %d", len(handler.records))
	}
	if _, ok := handler.records[0].attrs["output"]; ok {
		t.Fatalf("expected no output attr for empty output, got %#v", handler.records[0].attrs)
	}
}

func TestRecordFirstFailureKeepsFirstError(t *testing.T) {

	handler := &captureHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(prev)

	var mu sync.Mutex
	var firstErr error
	ctx := context.Background()

	first := newBuildFailure("swagger", phaseCompile, errors.New("first"), "out1")
	second := newBuildFailure("client-go", phaseCompile, errors.New("second"), "out2")

	recordFirstFailure(ctx, &mu, &firstErr, first)
	recordFirstFailure(ctx, &mu, &firstErr, second)

	if !errors.Is(firstErr, first) {
		t.Fatalf("expected first failure to be kept, got %v", firstErr)
	}
	if len(handler.records) != 1 {
		t.Fatalf("expected one log record, got %d", len(handler.records))
	}
	if handler.records[0].attrs["plugin"] != "swagger" {
		t.Fatalf("expected first plugin in log, got %#v", handler.records[0].attrs)
	}
}

func TestBuildFailureErrorMessageFormat(t *testing.T) {

	var buf bytes.Buffer
	failure := newBuildFailure("swagger", phaseCompile, errors.New("exit status 1"), "line one\nline two")

	if _, err := buf.WriteString(failure.Error()); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if strings.Contains(buf.String(), "line one") {
		t.Fatalf("output must not be embedded in error message")
	}
}
