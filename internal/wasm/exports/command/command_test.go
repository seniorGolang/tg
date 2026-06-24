package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCommandWorkDirRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := resolveCommandWorkDir(root, ".."); err == nil {
		t.Fatal("expected parent traversal to be rejected")
	}
}

func TestResolveCommandWorkDirAllowsChild(t *testing.T) {
	root := t.TempDir()
	got, err := resolveCommandWorkDir(root, "sub/dir")
	if err != nil {
		t.Fatalf("expected child workDir to be allowed: %v", err)
	}

	want := filepath.Join(root, "sub", "dir")
	if got != want {
		t.Fatalf("unexpected workDir: got %q want %q", got, want)
	}
}

func TestBuildCommandEnvUsesAllowlist(t *testing.T) {
	t.Setenv("TG_ALLOWED_SECRET", "ok")
	t.Setenv("TG_DENIED_SECRET", "bad")

	env := buildCommandEnv([]string{"TG_ALLOWED_SECRET"})
	joined := strings.Join(env, "\n")

	if !strings.Contains(joined, "TG_ALLOWED_SECRET=ok") {
		t.Fatalf("expected allowed env var in child env: %v", env)
	}
	if strings.Contains(joined, "TG_DENIED_SECRET=bad") {
		t.Fatalf("unexpected denied env var in child env: %v", env)
	}
}

func TestGetCommandResponseDeletesOnlyFinalResult(t *testing.T) {
	commandResponses.mu.Lock()
	commandResponses.data[1] = &CommandResponse{ExitCode: -2}
	commandResponses.data[2] = &CommandResponse{ExitCode: 0}
	commandResponses.mu.Unlock()
	t.Cleanup(func() {
		commandResponses.mu.Lock()
		delete(commandResponses.data, 1)
		delete(commandResponses.data, 2)
		commandResponses.mu.Unlock()
	})

	if _, ok := getCommandResponse(1); !ok {
		t.Fatal("expected running command response")
	}
	commandResponses.mu.RLock()
	_, runningStillStored := commandResponses.data[1]
	commandResponses.mu.RUnlock()
	if !runningStillStored {
		t.Fatal("running response must stay for polling")
	}

	if _, ok := getCommandResponse(2); !ok {
		t.Fatal("expected final command response")
	}
	commandResponses.mu.RLock()
	_, finalStillStored := commandResponses.data[2]
	commandResponses.mu.RUnlock()
	if finalStillStored {
		t.Fatal("final response must be deleted after read")
	}
}

func TestResolveCommandWorkDirHandlesAbsoluteRoot(t *testing.T) {
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveCommandWorkDir(root, ".")
	if err != nil {
		t.Fatalf("expected root workDir to be allowed: %v", err)
	}
	if got != root {
		t.Fatalf("unexpected root workDir: got %q want %q", got, root)
	}
}

func TestResolveCommandWorkDirRejectsSiblingWithSharedPrefix(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveCommandWorkDir(root, "../root-other"); err == nil {
		t.Fatal("expected sibling with shared prefix to be rejected")
	}
}
