// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package manifest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

const (
	packageNameSeparator = "/"
)

// FindPackage ищет пакет в каталоге по source/package согласно архитектуре.
func (m *manager) FindPackage(ctx context.Context, packageName string) (pkg *models.Package, manifest *models.Manifest, err error) {

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if err = m.ensureIndex(ctx); err != nil {
		return
	}

	source, packageNameOnly := m.parsePackageName(packageName)
	normalizedSource := ""
	if source != "" {
		normalizedSource = storage.NormalizeSource(source)
	}

	slog.Debug(i18n.Msg("FindPackage: parsed package name"), slog.String("packageName", packageName), slog.String("source", source), slog.String("packageNameOnly", packageNameOnly), slog.String("normalizedSource", normalizedSource))

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	cachedList, exists := m.index.byPackageName[packageNameOnly]
	if !exists || len(cachedList) == 0 {
		slog.Debug(i18n.Msg("FindPackage: package not found in index"), slog.String("packageNameOnly", packageNameOnly))
		return nil, nil, fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
	}

	sources := make([]string, 0, len(cachedList))
	for _, cached := range cachedList {
		sources = append(sources, cached.source)
	}
	slog.Debug(i18n.Msg("FindPackage: found packages in index"), slog.String("packageNameOnly", packageNameOnly), slog.Int("count", len(cachedList)), slog.Any("sources", sources))

	type match struct {
		pkg      *models.Package
		manifest *models.Manifest
		source   string
	}
	matches := make([]match, 0, len(cachedList))

	for _, cached := range cachedList {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}
		if normalizedSource != "" {
			normalizedCachedSource := storage.NormalizeSource(cached.source)
			if normalizedCachedSource != normalizedSource {
				slog.Debug(i18n.Msg("FindPackage: skipping cached manifest (source mismatch)"), slog.String("cached.source", cached.source), slog.String("normalizedCachedSource", normalizedCachedSource), slog.String("normalizedSource", normalizedSource))
				continue
			}
		}

		for i := range cached.manifest.Packages {
			if cached.manifest.Packages[i].Name == packageNameOnly {
				matches = append(matches, match{
					pkg:      &cached.manifest.Packages[i],
					manifest: cached.manifest,
					source:   cached.source,
				})
				break
			}
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil, fmt.Errorf(i18n.Msg("Package %s not found"), packageName)
	case 1:
		pkg = matches[0].pkg
		manifest = matches[0].manifest
		return
	default:
		conflictingSources := make([]string, 0, len(matches))
		for _, m := range matches {
			conflictingSources = append(conflictingSources, m.source)
		}
		return nil, nil, fmt.Errorf(i18n.Msg("Package %s found in multiple manifests")+":\n%s", packageName, strings.Join(conflictingSources, "\n"))
	}
}

// Формат: "source/package" (source может содержать "/"); последний сегмент — имя пакета.
func (m *manager) parsePackageName(packageName string) (source string, packageNameOnly string) {

	parts := strings.Split(packageName, packageNameSeparator)
	if len(parts) >= 3 {
		source = strings.Join(parts[:len(parts)-1], packageNameSeparator)
		packageNameOnly = parts[len(parts)-1]
	} else {
		packageNameOnly = packageName
	}

	return
}

// FindAllPackages ищет все пакеты с заданным именем во всех манифестах.
func (m *manager) FindAllPackages(ctx context.Context, packageName string) (packages []managers.PackageWithSource, err error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err = m.ensureIndex(ctx); err != nil {
		return
	}

	_, packageNameOnly := m.parsePackageName(packageName)

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	cachedList, exists := m.index.byPackageName[packageNameOnly]
	if !exists || len(cachedList) == 0 {
		packages = []managers.PackageWithSource{}
		return
	}

	packages = make([]managers.PackageWithSource, 0, len(cachedList))
	for _, cached := range cachedList {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		for i := range cached.manifest.Packages {
			if cached.manifest.Packages[i].Name == packageNameOnly {
				packages = append(packages, managers.PackageWithSource{
					Package:  &cached.manifest.Packages[i],
					Source:   cached.source,
					Manifest: cached.manifest,
				})
			}
		}
	}

	return
}
