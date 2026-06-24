package installation

import (
	"context"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/installer/contextkeys"
)

func TestResolveScriptSourceRejectsHTTPWithoutForce(t *testing.T) {
	m := &manager{}
	_, err := m.resolveScriptSource(context.Background(), "http://example.test/install.sh", t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected http script source to be rejected without force")
	}
}

func TestResolveScriptSourceAllowsHTTPWithForce(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextkeys.Force, true)
	m := &manager{}
	_, err := m.resolveScriptSource(ctx, "http://127.0.0.1:1/not-running.sh", t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected download to be attempted and fail, not rejected by policy")
	}
	if err.Error() == "HTTP script sources are not allowed without --force" {
		t.Fatal("expected force to bypass cleartext source policy")
	}
}
