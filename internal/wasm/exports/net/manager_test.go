package net

import (
	"context"
	"io"
	stdnet "net"
	"sync/atomic"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
	"github.com/seniorGolang/tg/v3/internal/wasm/stream"
)

type fakeStreamRegistry struct {
	closed atomic.Int32
}

func (f *fakeStreamRegistry) NewStream(context.Context, memory.Host, io.Reader, io.Writer, uint32) (uint32, error) {
	return 99, nil
}

func (f *fakeStreamRegistry) GetStreamState(uint32) (*stream.StreamState, bool) {
	return nil, false
}

func (f *fakeStreamRegistry) CloseStream(context.Context, memory.Host, uint32) {
	f.closed.Add(1)
}

func (f *fakeStreamRegistry) GetBufferPtr(uint32) (uint32, bool) {
	return 0, false
}

func (f *fakeStreamRegistry) GetReadBufferPtr(uint32) (uint32, bool) {
	return 0, false
}

func (f *fakeStreamRegistry) GetWriteBufferPtr(uint32) (uint32, bool) {
	return 0, false
}

func (f *fakeStreamRegistry) WaitReaderDone(uint32) {}

func TestDelConnClosesAssociatedStream(t *testing.T) {
	server, client := stdnet.Pipe()
	defer func() { _ = client.Close() }()

	registry := &fakeStreamRegistry{}
	hst := &host.Host{StreamRegistry: registry}
	nm := NewNetManager()

	connID := nm.StoreConnWithStream(context.Background(), hst, server)
	nm.DelConn(connID)

	if got := registry.closed.Load(); got != 1 {
		t.Fatalf("expected stream to be closed once, got %d", got)
	}
}
