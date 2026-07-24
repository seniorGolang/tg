// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHostSkipsMissingAgents(t *testing.T) {

	tgHome := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("TG_HOME", tgHome)
	t.Setenv("HOME", userHome)

	opts := Default()
	if err := InstallHost(opts); err != nil {
		t.Fatal(err)
	}

	canon := filepath.Join(tgHome, "skills", "host", "tg", "SKILL.md")
	if _, err := os.Stat(canon); err != nil {
		t.Fatal(err)
	}
	published := filepath.Join(userHome, ".agents", "skills", "tg")
	if _, err := os.Stat(published); !os.IsNotExist(err) {
		t.Fatalf("expected no publish without ~/.agents, err=%v", err)
	}
}

func TestInstallHostPublishesWhenAgentsExists(t *testing.T) {

	tgHome := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("TG_HOME", tgHome)
	t.Setenv("HOME", userHome)
	if err := os.MkdirAll(filepath.Join(userHome, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := InstallHost(Default()); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"tg", "tg-plugin"} {
		path := filepath.Join(userHome, ".agents", "skills", name, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	statePath := filepath.Join(tgHome, "skills", "host.yml")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatal(err)
	}

	if err := InstallHost(Default()); err != nil {
		t.Fatal(err)
	}
}

func TestInstallHostMkdir(t *testing.T) {

	tgHome := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("TG_HOME", tgHome)
	t.Setenv("HOME", userHome)

	opts := Default()
	opts.Mkdir = true
	if err := InstallHost(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(userHome, ".agents", "skills", "tg", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}
