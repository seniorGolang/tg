// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
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
	if err = packagePluginSkills(root, outDir, "1.0.0", &b); err != nil {
		t.Fatal(err)
	}
	if b.SkillsArchive == "" || len(b.Skills) != 1 {
		t.Fatalf("built=%+v", b)
	}
	if want := "skills/server/1.0.0/tgp-server"; b.Skills[0].Root != want {
		t.Fatalf("root=%s want=%s", b.Skills[0].Root, want)
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

// Skills и плагин версионируются одинаково, поэтому разные версии пакета не борются за одни пути.
func TestSkillDestinationsAreVersioned(t *testing.T) {

	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins", "astg")
	skillDir := filepath.Join(pluginDir, "skills", "tgp-contracts", "references")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		filepath.Join(pluginDir, "skills", "tgp-contracts", "SKILL.md"): "---\nname: tgp-contracts\n---\n",
		filepath.Join(skillDir, "annotations.md"):                       "ann",
	} {
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	outDir := t.TempDir()
	destinations := make(map[string][]string)
	for _, version := range []string{"1.0.7", "1.0.8"} {
		b := builtPlugin{Dir: "astg", Name: "astg", Info: plugin.Info{Description: "d"}}
		if err := packagePluginSkills(root, outDir, version, &b); err != nil {
			t.Fatal(err)
		}

		genPath, err := generateManifest(outDir, version, []builtPlugin{b})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(genPath)
		if err != nil {
			t.Fatal(err)
		}

		var manifest models.Manifest
		if err = yaml.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Packages) != 1 {
			t.Fatalf("packages=%+v", manifest.Packages)
		}
		for _, file := range manifest.Packages[0].Files {
			destinations[version] = append(destinations[version], file.Destination)
			if !strings.Contains(file.Destination, "/"+version+"/") {
				t.Fatalf("destination %q has no version %s", file.Destination, version)
			}
		}
		for _, spec := range manifest.Packages[0].Skills {
			if want := "skills/astg/" + version + "/tgp-contracts"; spec.Root != want {
				t.Fatalf("skill root=%s want=%s", spec.Root, want)
			}
		}
	}

	older := make(map[string]struct{}, len(destinations["1.0.7"]))
	for _, destination := range destinations["1.0.7"] {
		older[destination] = struct{}{}
	}
	for _, destination := range destinations["1.0.8"] {
		if _, clash := older[destination]; clash {
			t.Fatalf("destination %q shared between versions", destination)
		}
	}
}
