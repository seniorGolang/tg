// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package manifest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
	"github.com/seniorGolang/tg/v3/internal/installer/version"

	"gopkg.in/yaml.v3"
)

// loadManifestCascadeRecursive: requestedVersion — версия из URL (@v1.0.0); пустая значит не указана.
func (m *manager) loadManifestCascadeRecursive(ctx context.Context, manifestURL string, source string, force bool, requestedVersion string) (err error) {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	key := manifestURL
	if m.loadedURLs[key] {
		return
	}

	var manifest *models.Manifest
	if manifest, err = m.LoadManifest(ctx, manifestURL); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to load manifest %s: %w"), manifestURL, err)
	}

	if source == "" {
		source = storage.ExtractSourceFromManifestURL(manifestURL)
	}

	normalizedSource := storage.NormalizeSource(source)
	manifestDir := storage.GetManifestDir(m.scopeName, normalizedSource)

	m.loadedSources[normalizedSource] = true

	if !force {
		var existingVersion string
		if existingVersion, err = m.getExistingManifestVersion(manifestDir); err == nil {
			var newVersion models.Version
			if newVersion, err = version.Parse(manifest.Version); err == nil {
				var oldVersion models.Version
				if oldVersion, err = version.Parse(existingVersion); err == nil {
					comparison := version.Compare(newVersion, oldVersion)
					if comparison < 0 {
						if requestedVersion != "" {
							slog.Warn(
								i18n.Msg("Requested manifest version is older than existing version, using existing version"),
								slog.String("source", source),
								slog.String("requestedVersion", requestedVersion),
								slog.String("existingVersion", existingVersion),
							)
						}

						// Не перезаписываем более старой версией; refs загружаем из каталога, чтобы каскад был согласован.
						var existingManifest *models.Manifest
						if existingManifest, err = m.loadExistingManifest(manifestDir); err != nil {
							slog.Debug(i18n.Msg("Failed to load existing manifest for cascade processing"), slog.String("manifestDir", manifestDir), slog.Any("error", err))
							return
						}

						m.loadedURLs[key] = true
						if err = m.processManifestRefs(ctx, existingManifest, force); err != nil {
							return
						}

						return
					}
				}
			}
		}
	}

	if err = storage.EnsureDir(manifestDir); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "manifest directory", err)
	}

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var data []byte
	if data, err = yaml.Marshal(manifest); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to serialize manifest: %w"), err)
	}

	if err = os.WriteFile(manifestFile, data, 0600); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to save manifest: %w"), err)
	}

	m.loadedURLs[key] = true

	if err = m.processManifestRefs(ctx, manifest, force); err != nil {
		return
	}

	return
}

func (m *manager) getExistingManifestVersion(manifestDir string) (version string, err error) {

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var statErr error
	if _, statErr = os.Stat(manifestFile); os.IsNotExist(statErr) {
		return "", errors.New(i18n.Msg("Manifest does not exist"))
	}

	var data []byte
	if data, err = os.ReadFile(manifestFile); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to read manifest: %w"), err)
	}

	var manifest models.Manifest
	if err = yaml.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf(i18n.Msg("Failed to parse manifest: %w"), err)
	}

	return manifest.Version, nil
}

func (m *manager) loadExistingManifest(manifestDir string) (manifest *models.Manifest, err error) {

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var statErr error
	if _, statErr = os.Stat(manifestFile); os.IsNotExist(statErr) {
		return nil, errors.New(i18n.Msg("Manifest does not exist"))
	}

	var data []byte
	if data, err = os.ReadFile(manifestFile); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to read manifest: %w"), err)
	}

	manifest = &models.Manifest{}
	if err = yaml.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf(i18n.Msg("Failed to parse manifest: %w"), err)
	}

	return
}

func (m *manager) processManifestRefs(ctx context.Context, manifest *models.Manifest, force bool) (err error) {

	for _, ref := range manifest.Manifests {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var parsedURI uri.URI
		var parseErr error
		if parsedURI, parseErr = uri.New(ref.URL); parseErr != nil {
			slog.Warn(i18n.Msg("Failed to parse manifest URL, skipping"), slog.String("url", ref.URL), slog.Any("error", parseErr))
			continue
		}

		packageName := parsedURI.Package()
		if packageName != "" {
			slog.Warn(i18n.Msg("Manifest URL in manifests field contains package specification, skipping"), slog.String("url", ref.URL), slog.String("source", parsedURI.Source()), slog.String("package", packageName))
			continue
		}

		refSource := parsedURI.Source()
		requestedVersion := parsedURI.Version().Original

		var refManifestURL string
		if refManifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
			slog.Error(i18n.Msg("Failed to get manifest"), slog.String("source", refSource), slog.String("url", ref.URL), slog.Any("error", err))
			return fmt.Errorf(i18n.Msg("Failed to get manifest for %s: %w"), refSource, err)
		}

		if err = m.loadManifestCascadeRecursive(ctx, refManifestURL, refSource, force, requestedVersion); err != nil {
			return
		}
	}

	return
}
