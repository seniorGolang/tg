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

// loadManifestCascadeRecursive рекурсивно загружает манифест и связанные с учётом версий.
// requestedVersion - версия, явно указанная в URL (например, @v1.0.0). Если пустая, версия не была явно указана.
func (m *manager) loadManifestCascadeRecursive(ctx context.Context, manifestURL string, source string, force bool, requestedVersion string) (err error) {

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	key := manifestURL
	if m.loadedURLs[key] {
		return
	}

	var manifest *models.Manifest
	if manifest, err = m.LoadManifest(ctx, manifestURL); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to load manifest %s: %w"), manifestURL, err)
		return
	}

	if source == "" {
		source = storage.ExtractSourceFromManifestURL(manifestURL)
	}

	normalizedSource := storage.NormalizeSource(source)
	manifestDir := storage.GetManifestDir(m.scopeName, normalizedSource)

	// Отмечаем source как загруженный
	m.loadedSources[normalizedSource] = true

	// Если force=false, проверяем версию и пропускаем, если новая версия меньше
	if !force {
		var existingVersion string
		if existingVersion, err = m.getExistingManifestVersion(manifestDir); err == nil {
			var newVersion models.Version
			if newVersion, err = version.Parse(manifest.Version); err == nil {
				var oldVersion models.Version
				if oldVersion, err = version.Parse(existingVersion); err == nil {
					comparison := version.Compare(newVersion, oldVersion)
					if comparison < 0 {
						// Если версия была явно указана в URL, выводим предупреждение
						if requestedVersion != "" {
							slog.Warn(
								i18n.Msg("Requested manifest version is older than existing version, using existing version"),
								slog.String("source", source),
								slog.String("requestedVersion", requestedVersion),
								slog.String("existingVersion", existingVersion),
							)
						}

						// Не перезаписываем манифест более старой версией; вложенные манифесты (refs) всё равно
						// загружаем из уже сохранённого в каталоге файла, чтобы каскад оставался согласованным.
						// При пропуске используем существующий манифест из каталога для обработки вложенных манифестов
						var existingManifest *models.Manifest
						if existingManifest, err = m.loadExistingManifest(manifestDir); err != nil {
							slog.Debug(i18n.Msg("Failed to load existing manifest for cascade processing"), slog.String("manifestDir", manifestDir), slog.Any("error", err))
							return
						}

						// Отмечаем URL как загруженный
						m.loadedURLs[key] = true

						// Обрабатываем вложенные манифесты из существующего манифеста
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
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "manifest directory", err)
		return
	}

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var data []byte
	if data, err = yaml.Marshal(manifest); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to serialize manifest: %w"), err)
		return
	}

	if err = os.WriteFile(manifestFile, data, 0600); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to save manifest: %w"), err)
		return
	}

	m.loadedURLs[key] = true

	// Обрабатываем вложенные манифесты
	if err = m.processManifestRefs(ctx, manifest, force); err != nil {
		return
	}

	return
}

func (m *manager) getExistingManifestVersion(manifestDir string) (version string, err error) {

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var statErr error
	if _, statErr = os.Stat(manifestFile); os.IsNotExist(statErr) {
		err = errors.New(i18n.Msg("Manifest does not exist"))
		return
	}

	var data []byte
	if data, err = os.ReadFile(manifestFile); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read manifest: %w"), err)
		return
	}

	var manifest models.Manifest
	if err = yaml.Unmarshal(data, &manifest); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse manifest: %w"), err)
		return
	}

	version = manifest.Version
	return
}

// loadExistingManifest загружает существующий манифест из каталога.
func (m *manager) loadExistingManifest(manifestDir string) (manifest *models.Manifest, err error) {

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var statErr error
	if _, statErr = os.Stat(manifestFile); os.IsNotExist(statErr) {
		err = errors.New(i18n.Msg("Manifest does not exist"))
		return
	}

	var data []byte
	if data, err = os.ReadFile(manifestFile); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read manifest: %w"), err)
		return
	}

	manifest = &models.Manifest{}
	if err = yaml.Unmarshal(data, manifest); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse manifest: %w"), err)
		return
	}

	return
}

// processManifestRefs обрабатывает вложенные манифесты из поля manifests.
func (m *manager) processManifestRefs(ctx context.Context, manifest *models.Manifest, force bool) (err error) {

	for _, ref := range manifest.Manifests {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
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
			err = fmt.Errorf(i18n.Msg("Failed to get manifest for %s: %w"), refSource, err)
			return
		}

		if err = m.loadManifestCascadeRecursive(ctx, refManifestURL, refSource, force, requestedVersion); err != nil {
			return
		}
	}

	return
}
