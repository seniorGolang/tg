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

func generateManifest(outDir string, version string, built []builtPlugin) (genPath string, err error) {

	slog.Debug("generating manifest", "outDir", outDir, "version", version, "packages", len(built))

	var absOut string
	if absOut, err = filepath.Abs(outDir); err != nil {
		return
	}

	packages := make([]models.Package, 0, len(built))
	for _, b := range built {
		url := "file://" + filepath.Join(absOut, b.Name+".tgp")
		dest := "plugins/" + b.Dir + "/" + version + "/" + b.Name + ".tgp"
		packages = append(packages, models.Package{
			Name:         b.Name,
			Descr:        b.Info.Description,
			Dependencies: b.Info.Dependencies,
			Downloads:    []models.PlatformDownload{{URL: url}},
			Files: []models.FileInstallation{{
				File:        b.Name + ".tgp",
				Destination: dest,
				Checksum:    b.Checksum,
			}},
		})
	}

	gen := models.Manifest{
		Version:  version,
		Packages: packages,
	}

	var data []byte
	if data, err = yaml.Marshal(&gen); err != nil {
		return
	}

	genPath = filepath.Join(outDir, ".manifest.generated.yml")
	return genPath, os.WriteFile(genPath, data, 0600)
}
