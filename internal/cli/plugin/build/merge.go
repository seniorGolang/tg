// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func mergeAndWrite(genPath string, overridePath string, outDir string) (err error) {

	slog.Debug("merging manifest", "generated", genPath, "override", overridePath)

	var data []byte
	if data, err = os.ReadFile(genPath); err != nil {
		return
	}

	var gen models.Manifest
	if err = yaml.Unmarshal(data, &gen); err != nil {
		return
	}

	result := &gen
	ovData, readErr := os.ReadFile(overridePath)
	if readErr == nil {
		var ov models.Manifest
		if err = yaml.Unmarshal(ovData, &ov); err != nil {
			return
		}
		result = mergeManifests(&gen, &ov)
	}

	var outData []byte
	if outData, err = yaml.Marshal(result); err != nil {
		return
	}

	manifestPath := filepath.Join(outDir, "manifest.yml")
	if err = os.WriteFile(manifestPath, outData, 0600); err != nil {
		return
	}
	slog.Debug("manifest written", "path", manifestPath)

	return os.Remove(genPath)
}

func mergeManifests(gen *models.Manifest, ov *models.Manifest) (out *models.Manifest) {

	genByName := make(map[string]*models.Package)
	for i := range gen.Packages {
		p := &gen.Packages[i]
		genByName[p.Name] = p
	}
	ovByName := make(map[string]*models.Package)
	for i := range ov.Packages {
		p := &ov.Packages[i]
		ovByName[p.Name] = p
	}

	var order []string
	seen := make(map[string]bool)
	for _, p := range ov.Packages {
		if !seen[p.Name] {
			seen[p.Name] = true
			order = append(order, p.Name)
		}
	}
	for _, p := range gen.Packages {
		if !seen[p.Name] {
			seen[p.Name] = true
			order = append(order, p.Name)
		}
	}

	version := gen.Version
	if ov.Version != "" {
		version = ov.Version
	}

	packages := make([]models.Package, 0, len(order))
	for _, name := range order {
		g := genByName[name]
		o := ovByName[name]
		var p models.Package
		if g != nil {
			p = *g
		}
		if o != nil {
			mergePackage(&p, o)
		}
		p.Name = name
		packages = append(packages, p)
	}

	return &models.Manifest{Version: version, Packages: packages}
}

func mergePackage(dst *models.Package, src *models.Package) {

	if src.Descr != "" {
		dst.Descr = src.Descr
	}
	if src.Hidden {
		dst.Hidden = true
	}
	if src.Alias != "" {
		dst.Alias = src.Alias
	}
	if len(src.Downloads) > 0 {
		dst.Downloads = src.Downloads
	}
	if len(src.Files) > 0 {
		dst.Files = src.Files
	}
	if src.Scripts != nil {
		dst.Scripts = src.Scripts
	}
	if len(src.Dependencies) > 0 {
		dst.Dependencies = src.Dependencies
	}
}
