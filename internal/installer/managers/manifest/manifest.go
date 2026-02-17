// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package manifest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
	"github.com/seniorGolang/tg/v3/internal/installer/version"

	"gopkg.in/yaml.v3"
)

const (
	protocolFile = "file://"
)

type cachedManifest struct {
	manifest *models.Manifest
	path     string
	source   string
	modTime  time.Time
}

type manifestIndex struct {
	byPackageName map[string][]*cachedManifest
	bySource      map[string][]*cachedManifest
	allManifests  []*cachedManifest
	lastUpdate    time.Time
	mu            sync.RWMutex
}

type manager struct {
	scopeName     string
	loadedURLs    map[string]bool
	loadedSources map[string]bool
	index         *manifestIndex
	indexOnce     sync.Once
}

func NewManager(scopeName string) managers.ManifestManager {
	return &manager{
		scopeName:     scopeName,
		loadedURLs:    make(map[string]bool),
		loadedSources: make(map[string]bool),
		index: &manifestIndex{
			byPackageName: make(map[string][]*cachedManifest),
			bySource:      make(map[string][]*cachedManifest),
			allManifests:  make([]*cachedManifest, 0),
		},
	}
}

func (m *manager) ensureIndex(ctx context.Context) (err error) {

	m.indexOnce.Do(func() {
		err = m.ReloadIndex(ctx)
	})
	return
}

func (m *manager) ReloadIndex(ctx context.Context) (err error) {

	m.index.mu.Lock()
	defer m.index.mu.Unlock()

	catalogDir := storage.GetCatalogDir(m.scopeName)
	if _, statErr := os.Stat(catalogDir); os.IsNotExist(statErr) {
		return
	}

	newIndex := &manifestIndex{
		byPackageName: make(map[string][]*cachedManifest),
		bySource:      make(map[string][]*cachedManifest),
		allManifests:  make([]*cachedManifest, 0),
		lastUpdate:    time.Now(),
	}

	err = filepath.Walk(catalogDir, func(path string, info os.FileInfo, walkErr error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() || (info.Name() != storage.ManifestFileName && info.Name() != storage.ManifestFileNameYAML) {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		var manifest models.Manifest
		if unmarshalErr := yaml.Unmarshal(data, &manifest); unmarshalErr != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(catalogDir, filepath.Dir(path))
		if relErr != nil {
			return nil
		}

		source := storage.ExtractSourceFromNormalizedPath(relPath)
		cached := &cachedManifest{
			manifest: &manifest,
			path:     path,
			source:   source,
			modTime:  info.ModTime(),
		}

		newIndex.allManifests = append(newIndex.allManifests, cached)
		newIndex.bySource[source] = append(newIndex.bySource[source], cached)

		for i := range manifest.Packages {
			packageName := manifest.Packages[i].Name
			newIndex.byPackageName[packageName] = append(newIndex.byPackageName[packageName], cached)
		}

		return nil
	})

	if err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to load manifest index: %w"), err)
		return
	}

	m.index = newIndex
	return
}

func (m *manager) LoadManifest(ctx context.Context, url string) (manifest *models.Manifest, err error) {

	var content []byte
	if content, err = m.downloadManifest(ctx, url); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
		return
	}

	manifest = &models.Manifest{}
	if err = yaml.Unmarshal(content, manifest); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse manifest: %w"), err)
		return
	}

	if err = m.ValidateManifest(ctx, manifest); err != nil {
		err = fmt.Errorf(i18n.Msg("Manifest validation failed: %w"), err)
		return
	}

	return
}

func (m *manager) LoadManifestCascade(ctx context.Context, manifestURL string, source string, force bool) (loadedSources map[string]bool, err error) {

	m.loadedURLs = make(map[string]bool)
	m.loadedSources = make(map[string]bool)
	if err = m.loadManifestCascadeRecursive(ctx, manifestURL, source, force, ""); err != nil {
		return
	}

	loadedSources = m.loadedSources
	return
}

func (m *manager) UpdateManifest(ctx context.Context, source string, force bool) (err error) {

	slog.Debug(i18n.Msg("Updating manifest"), slog.String("source", source), slog.Bool("force", force))

	var manifestURL string
	if manifestURL, err = m.buildManifestURLForUpdate(ctx, source); err != nil {
		slog.Error(i18n.Msg("Failed to get manifest"), slog.String("source", source), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to get manifest: %w"), err)
		return
	}

	slog.Debug(i18n.Msg("Built manifest URL for update"), slog.String("source", source), slog.String("manifestURL", manifestURL))

	normalizedSource := storage.NormalizeSource(source)
	manifestDir := storage.GetManifestDir(m.scopeName, normalizedSource)

	if !force {
		var existingVersion string
		if existingVersion, err = m.getExistingManifestVersion(manifestDir); err == nil {
			var newManifest *models.Manifest
			if newManifest, err = m.LoadManifest(ctx, manifestURL); err != nil {
				slog.Error(i18n.Msg("Failed to load manifest for version comparison"), slog.String("source", source), slog.String("manifestURL", manifestURL), slog.Any("error", err))
				err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
				return
			}

			var newVersion models.Version
			if newVersion, err = version.Parse(newManifest.Version); err != nil {
				slog.Error(i18n.Msg("Invalid new manifest version format"), slog.String("source", source), slog.String("manifestURL", manifestURL), slog.String("version", newManifest.Version), slog.Any("error", err))
				err = fmt.Errorf(i18n.Msg("Invalid new manifest version format: %w"), err)
				return
			}

			var oldVersion models.Version
			if oldVersion, err = version.Parse(existingVersion); err == nil {
				comparison := version.Compare(newVersion, oldVersion)
				if comparison < 0 {
					slog.Debug(i18n.Msg("Skipping manifest update: new version is older"), slog.String("source", source), slog.String("existingVersion", existingVersion), slog.String("newVersion", newManifest.Version))
					return
				}
			}
		}
	}

	if _, err = m.LoadManifestCascade(ctx, manifestURL, source, force); err != nil {
		slog.Error(i18n.Msg("Failed to load manifest cascade"), slog.String("source", source), slog.String("manifestURL", manifestURL), slog.Any("error", err))
		return
	}

	slog.Debug(i18n.Msg("Successfully updated manifest"), slog.String("source", source), slog.String("manifestURL", manifestURL))
	return
}

func (m *manager) ListPackages(ctx context.Context) (packages []models.Package, err error) {

	if err = m.ensureIndex(ctx); err != nil {
		packages = nil
		return
	}

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	packages = make([]models.Package, 0, len(m.index.allManifests)*2)
	for _, cached := range m.index.allManifests {
		for i := range cached.manifest.Packages {
			if cached.manifest.Packages[i].Hidden {
				continue
			}
			packages = append(packages, cached.manifest.Packages[i])
		}
	}

	return
}

func (m *manager) ListPackagesFromSources(ctx context.Context, sources map[string]bool) (packages []models.Package, err error) {

	if err = m.ensureIndex(ctx); err != nil {
		packages = nil
		return
	}

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	packages = make([]models.Package, 0)
	for _, cached := range m.index.allManifests {
		normalizedCachedSource := storage.NormalizeSource(cached.source)
		if !sources[normalizedCachedSource] {
			continue
		}
		for i := range cached.manifest.Packages {
			if cached.manifest.Packages[i].Hidden {
				continue
			}
			packages = append(packages, cached.manifest.Packages[i])
		}
	}

	return
}

func (m *manager) SearchPackages(ctx context.Context, query string) (packages []models.Package, err error) {

	var allPackages []models.Package
	if allPackages, err = m.ListPackages(ctx); err != nil {
		packages = nil
		return
	}

	query = strings.ToLower(query)
	packages = make([]models.Package, 0)

	for _, pkg := range allPackages {
		if strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.Descr), query) {
			packages = append(packages, pkg)
		}
	}

	return
}

func (m *manager) ValidateManifest(ctx context.Context, manifest *models.Manifest) (err error) {

	if manifest.Version == "" {
		err = errors.New(i18n.Msg("Manifest version not specified"))
		return
	}

	var v models.Version
	if v, err = version.Parse(manifest.Version); err != nil {
		err = fmt.Errorf(i18n.Msg("Invalid version format: %w"), err)
		return
	}
	_ = v

	return
}

func (m *manager) GetCatalog(ctx context.Context) (manifests []managers.ManifestInfo, err error) {

	dbFile := storage.GetPackagesDBFile(m.scopeName)
	sourceSet := make(map[string]bool)

	if _, statErr := os.Stat(dbFile); statErr == nil {
		data, readErr := os.ReadFile(dbFile)
		if readErr == nil {
			var db models.InstallationDatabase
			//nolint:musttag // структура InstallationDatabase и все вложенные структуры имеют теги yaml в пакете models
			if unmarshalErr := yaml.Unmarshal(data, &db); unmarshalErr == nil {
				for _, inst := range db.Installed {
					if inst.Source != "" {
						sourceSet[inst.Source] = true
					}
				}
			}
		}
	}

	if len(sourceSet) == 0 {
		manifests = []managers.ManifestInfo{}
		return
	}

	if err = m.ensureIndex(ctx); err != nil {
		manifests = nil
		return
	}

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	manifests = make([]managers.ManifestInfo, 0, len(sourceSet))
	for source := range sourceSet {
		cachedList, exists := m.index.bySource[source]
		if !exists || len(cachedList) == 0 {
			continue
		}

		latestCached := cachedList[0]
		for _, cached := range cachedList[1:] {
			version1, parseErr1 := version.Parse(latestCached.manifest.Version)
			version2, parseErr2 := version.Parse(cached.manifest.Version)
			if parseErr1 == nil && parseErr2 == nil {
				if version.Compare(version2, version1) > 0 {
					latestCached = cached
				}
			}
		}

		manifests = append(manifests, managers.ManifestInfo{
			URL:      source,
			Version:  latestCached.manifest.Version,
			LoadedAt: latestCached.modTime.String(),
		})
	}

	return
}

func (m *manager) GetManifestVersion(ctx context.Context, url string) (versionStr string, err error) {

	if err = m.ensureIndex(ctx); err != nil {
		versionStr = ""
		return
	}

	source := storage.ExtractSourceFromManifestURL(url)
	normalizedSource := storage.NormalizeSource(source)

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	cachedList, exists := m.index.bySource[normalizedSource]
	if !exists || len(cachedList) == 0 {
		err = errors.New(i18n.Msg("Manifest not found"))
		return
	}

	latestCached := cachedList[0]
	for _, cached := range cachedList[1:] {
		version1, parseErr1 := version.Parse(latestCached.manifest.Version)
		version2, parseErr2 := version.Parse(cached.manifest.Version)
		if parseErr1 == nil && parseErr2 == nil {
			if version.Compare(version2, version1) > 0 {
				latestCached = cached
			}
		}
	}

	versionStr = latestCached.manifest.Version
	return
}

func (m *manager) GetAllManifests(ctx context.Context) (manifests []managers.ManifestWithSource, err error) {

	if err = m.ensureIndex(ctx); err != nil {
		manifests = nil
		return
	}

	m.index.mu.RLock()
	defer m.index.mu.RUnlock()

	manifests = make([]managers.ManifestWithSource, 0, len(m.index.allManifests))
	for _, cached := range m.index.allManifests {
		manifests = append(manifests, managers.ManifestWithSource{
			Manifest: cached.manifest,
			Source:   cached.source,
			Path:     cached.path,
		})
	}

	return
}

func (m *manager) CompareVersions(ctx context.Context, v1 string, v2 string) (result int, err error) {

	var version1 models.Version
	if version1, err = version.Parse(v1); err != nil {
		err = fmt.Errorf(i18n.Msg("Invalid version format v1: %w"), err)
		return
	}

	var version2 models.Version
	if version2, err = version.Parse(v2); err != nil {
		err = fmt.Errorf(i18n.Msg("Invalid version format v2: %w"), err)
		return
	}

	return version.Compare(version1, version2), nil
}

func (m *manager) buildManifestURLForUpdate(ctx context.Context, source string) (manifestURL string, err error) {

	slog.Debug(i18n.Msg("Building manifest URL for update"), slog.String("source", source))

	var parsedURI uri.URI
	if parsedURI, err = uri.New(source); err != nil {
		err = fmt.Errorf("failed to parse source URL: %w", err)
		return
	}

	if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
		slog.Error(i18n.Msg("Failed to get manifest"), slog.String("source", source), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to get manifest: %w"), err)
		return
	}

	slog.Debug(i18n.Msg("Built manifest URL"), slog.String("source", source), slog.String("manifestURL", manifestURL))
	return
}

func (m *manager) downloadManifest(ctx context.Context, url string) (content []byte, err error) {

	slog.Debug(i18n.Msg("Downloading manifest"), slog.String("url", url))

	if strings.HasPrefix(url, protocolFile) {
		path := strings.TrimPrefix(url, protocolFile)
		slog.Debug(i18n.Msg("Loading manifest from file"), slog.String("path", path))
		return os.ReadFile(path)
	}

	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil); err != nil {
		slog.Error(fmt.Sprintf(i18n.Msg("Failed to create %s"), "HTTP request for manifest"), slog.String("url", url), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "request", err)
		return
	}

	client := &http.Client{}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		slog.Error(i18n.Msg("Failed to execute HTTP request for manifest"), slog.String("url", url), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
		return
	}
	defer resp.Body.Close()

	slog.Debug(i18n.Msg("Received response for manifest"), slog.String("url", url), slog.Int("statusCode", resp.StatusCode), slog.String("status", resp.Status))

	if resp.StatusCode != http.StatusOK {
		var body []byte
		body, _ = io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}

		slog.Error(i18n.Msg("Manifest download returned non-OK status"), slog.String("url", url), slog.Int("statusCode", resp.StatusCode), slog.String("status", resp.Status), slog.String("responseBody", bodyStr))
		err = fmt.Errorf(i18n.Msg("Unexpected status code: %d"), resp.StatusCode)
		return
	}

	if content, err = io.ReadAll(resp.Body); err != nil {
		slog.Error(i18n.Msg("Failed to read manifest response body"), slog.String("url", url), slog.Int("statusCode", resp.StatusCode), slog.Any("error", err))
		err = fmt.Errorf(i18n.Msg("Failed to read response body: %w"), err)
		return
	}

	slog.Debug(i18n.Msg("Successfully downloaded manifest"), slog.String("url", url), slog.Int("size", len(content)))
	return
}
