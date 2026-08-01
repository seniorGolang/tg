// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package download

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadWithProgressWritesFullFile(t *testing.T) {

	const body = "hello-archive"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "13")
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "file.tar.gz")
	m := &manager{httpClient: &http.Client{}}
	var lastPercent int
	if err := m.DownloadWithProgress(context.Background(), server.URL, dest, func(percent int) {
		lastPercent = percent
	}); err != nil {
		t.Fatalf("DownloadWithProgress: %v", err)
	}
	if lastPercent != 100 {
		t.Fatalf("lastPercent = %d, want 100", lastPercent)
	}

	var got []byte
	var err error
	if got, err = os.ReadFile(dest); err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != body {
		t.Fatalf("content = %q, want %q", got, body)
	}
}

func TestDownloadWithProgressRejectsShortBody(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "short")
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "file.tar.gz")
	m := &manager{httpClient: &http.Client{}}
	if err := m.DownloadWithProgress(context.Background(), server.URL, dest, nil); err == nil {
		t.Fatal("expected Content-Length mismatch error")
	}
}

func TestDownloadWithProgressReturnsHTTPError(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "missing.tar.gz")
	m := &manager{httpClient: &http.Client{}}
	if err := m.DownloadWithProgress(context.Background(), server.URL, dest, nil); err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestDownloadWithProgressReturnsCancelError(t *testing.T) {

	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	dest := filepath.Join(t.TempDir(), "cancel.tar.gz")
	m := &manager{httpClient: &http.Client{}}

	errChan := make(chan error, 1)
	go func() {
		errChan <- m.DownloadWithProgress(ctx, server.URL, dest, nil)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}
	cancel()

	var err error
	select {
	case err = <-errChan:
	case <-time.After(2 * time.Second):
		t.Fatal("DownloadWithProgress did not return after cancel")
	}
	if err == nil {
		t.Fatal("expected non-nil error after cancel")
	}
}
