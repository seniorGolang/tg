// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/plugin"
)

func TestCollectAndPackageSkills(t *testing.T) {

	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins", "server")
	skillDir := filepath.Join(pluginDir, "skills", "tgp-server")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: tgp-server\ndescription: x\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := collectSkills(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "tgp-server" {
		t.Fatalf("entries=%+v", entries)
	}

	outDir := t.TempDir()
	b := builtPlugin{Dir: "server", Name: "server", Info: plugin.Info{Description: "d"}}
	if err = packagePluginSkills(root, outDir, &b); err != nil {
		t.Fatal(err)
	}
	if b.SkillsArchive == "" || len(b.Skills) != 1 {
		t.Fatalf("built=%+v", b)
	}
	if _, err = os.Stat(filepath.Join(outDir, b.SkillsArchive)); err != nil {
		t.Fatal(err)
	}

	genPath, err := generateManifest(outDir, "1.0.0", []builtPlugin{b})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, part := range []string{"skills:", "tgp-server", "server-skills.tar.gz"} {
		if !strings.Contains(text, part) {
			t.Fatalf("manifest missing %q:\n%s", part, text)
		}
	}
}
