// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTargets(t *testing.T) {

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty defaults to agents", raw: "", want: []string{TargetAgents}},
		{name: "single", raw: "cursor", want: []string{TargetCursor}},
		{name: "many", raw: "agents, cursor,claude", want: []string{TargetAgents, TargetCursor, TargetClaude}},
		{name: "spaces only defaults", raw: " , ", want: []string{TargetAgents}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTargets(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("len=%d want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v want %v", got, tt.want)
				}
			}
		})
	}
}

func TestResolveTargetsSkipsMissingWithoutMkdir(t *testing.T) {

	home := t.TempDir()
	targets, skipped, err := ResolveTargets(home, []string{TargetAgents, TargetCursor}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 0 {
		t.Fatalf("expected no targets, got %v", targets)
	}
	if len(skipped) != 2 {
		t.Fatalf("expected 2 skipped, got %v", skipped)
	}
}

func TestResolveTargetsCreatesWithMkdir(t *testing.T) {

	home := t.TempDir()
	targets, skipped, err := ResolveTargets(home, []string{TargetAgents}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped: %v", skipped)
	}
	if len(targets) != 1 || targets[0].Name != TargetAgents {
		t.Fatalf("unexpected targets: %+v", targets)
	}
	if _, err = os.Stat(filepath.Join(home, ".agents")); err != nil {
		t.Fatal(err)
	}
}

func TestResolveTargetsUnknown(t *testing.T) {

	_, _, err := ResolveTargets(t.TempDir(), []string{"unknown"}, false)
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
}

func TestPublishAndRemove(t *testing.T) {

	src := t.TempDir()
	skillDir := filepath.Join(src, "tgp-demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: tgp-demo\ndescription: demo\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	destRoot := t.TempDir()
	dest := filepath.Join(destRoot, "tgp-demo")
	if err := Publish(skillDir, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	if err := Publish(skillDir, dest); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("expected removed, err=%v", err)
	}
}

func TestActivateSkipsMissingAgents(t *testing.T) {

	home := t.TempDir()
	src := t.TempDir()
	skillDir := filepath.Join(src, "tgp-demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	states, skipped, err := Activate(home, []Root{{Name: "tgp-demo", Path: skillDir}}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || skipped[0] != TargetAgents {
		t.Fatalf("skipped=%v", skipped)
	}
	if len(states) != 1 || len(states[0].Published) != 0 {
		t.Fatalf("states=%+v", states)
	}
}

func TestActivatePublishesWhenRootExists(t *testing.T) {

	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	skillDir := filepath.Join(src, "tgp-demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	states, skipped, err := Activate(home, []Root{{Name: "tgp-demo", Path: skillDir}}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped=%v", skipped)
	}
	want := filepath.Join(home, ".agents", "skills", "tgp-demo")
	if len(states) != 1 || len(states[0].Published) != 1 || states[0].Published[0] != want {
		t.Fatalf("states=%+v", states)
	}
	if _, err = os.Stat(filepath.Join(want, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestActivateDisabled(t *testing.T) {

	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".agents"), 0755)
	src := t.TempDir()
	skillDir := filepath.Join(src, "tgp-demo")
	_ = os.MkdirAll(skillDir, 0755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0644)

	opts := Default()
	opts.Enabled = false
	states, skipped, err := Activate(home, []Root{{Name: "tgp-demo", Path: skillDir}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 || len(states[0].Published) != 0 {
		t.Fatalf("states=%+v skipped=%v", states, skipped)
	}
}

func TestScan(t *testing.T) {

	prefix := t.TempDir()
	skillDir := filepath.Join(prefix, "skills", "server", "tgp-server")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(prefix, "skills", "server", "empty"), 0755)

	roots, err := Scan(prefix, "server")
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0].Name != "tgp-server" {
		t.Fatalf("roots=%+v", roots)
	}
}

func TestDeactivate(t *testing.T) {

	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, ".agents"), 0755)
	src := t.TempDir()
	skillDir := filepath.Join(src, "tgp-demo")
	_ = os.MkdirAll(skillDir, 0755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0644)

	states, _, err := Activate(home, []Root{{Name: "tgp-demo", Path: skillDir}}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if err = Deactivate(states); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(states[0].Published[0]); !os.IsNotExist(err) {
		t.Fatalf("expected removed published skill")
	}
}
