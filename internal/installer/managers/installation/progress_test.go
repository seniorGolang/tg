// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package installation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/seniorGolang/tg/v3/internal/ui"
)

const (
	partialMarker  = "partial"
	completeMarker = "complete"
)

type delayedDownload struct {
	delay     time.Duration
	returnErr error
}

func (d *delayedDownload) Download(ctx context.Context, url string, destination string) (err error) {

	return nil
}

func (d *delayedDownload) DownloadWithProgress(ctx context.Context, url string, destination string, onProgress func(percent int)) (err error) {

	if err = os.WriteFile(destination, []byte(partialMarker), 0o644); err != nil {
		return err
	}
	if onProgress != nil {
		onProgress(50)
	}

	timer := time.NewTimer(d.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	if err = os.WriteFile(destination, []byte(completeMarker), 0o644); err != nil {
		return err
	}
	if onProgress != nil {
		onProgress(100)
	}
	return d.returnErr
}

func TestDownloadWithProgressCompletesFileBeforeReturn(t *testing.T) {

	dest := filepath.Join(t.TempDir(), "archive.tar.gz")
	m := &manager{
		downloadManager: &delayedDownload{delay: 50 * time.Millisecond},
	}
	bar := ui.NewProgressBar("test", 100, 8)

	if err := m.downloadWithProgress(context.Background(), "https://example.test/archive.tar.gz", dest, bar); err != nil {
		t.Fatalf("downloadWithProgress: %v", err)
	}

	var got []byte
	var err error
	if got, err = os.ReadFile(dest); err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != completeMarker {
		t.Fatalf("content = %q, want %q", got, completeMarker)
	}
}

func TestDownloadWithProgressReturnsDownloadError(t *testing.T) {

	dest := filepath.Join(t.TempDir(), "archive.tar.gz")
	wantErr := errors.New("download failed")
	m := &manager{
		downloadManager: &delayedDownload{delay: 10 * time.Millisecond, returnErr: wantErr},
	}
	bar := ui.NewProgressBar("test", 100, 8)

	if err := m.downloadWithProgress(context.Background(), "https://example.test/archive.tar.gz", dest, bar); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestDownloadWithProgressCancelReturnsContextError(t *testing.T) {

	dest := filepath.Join(t.TempDir(), "archive.tar.gz")
	m := &manager{
		downloadManager: &delayedDownload{delay: 2 * time.Second},
	}
	bar := ui.NewProgressBar("test", 100, 8)

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.downloadWithProgress(ctx, "https://example.test/archive.tar.gz", dest, bar)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	var err error
	select {
	case err = <-errChan:
	case <-time.After(2 * time.Second):
		t.Fatal("downloadWithProgress did not return after cancel")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
