// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package managers

import (
	"context"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// ManifestManager управляет манифестами.
type ManifestManager interface {
	LoadManifest(ctx context.Context, url string) (manifest *models.Manifest, err error)
	LoadManifestCascade(ctx context.Context, manifestURL string, source string, force bool) (loadedSources map[string]bool, err error)
	UpdateManifest(ctx context.Context, source string, force bool) (err error)
	ReloadIndex(ctx context.Context) (err error)
	FindPackage(ctx context.Context, packageName string) (pkg *models.Package, manifest *models.Manifest, err error)
	FindAllPackages(ctx context.Context, packageName string) (packages []PackageWithSource, err error)
	ListPackages(ctx context.Context) (packages []models.Package, err error)
	ListPackagesFromSources(ctx context.Context, sources map[string]bool) (packages []models.Package, err error)
	SearchPackages(ctx context.Context, query string) (packages []models.Package, err error)
	ValidateManifest(ctx context.Context, manifest *models.Manifest) (err error)
	GetCatalog(ctx context.Context) (manifests []ManifestInfo, err error)
	GetManifestVersion(ctx context.Context, url string) (version string, err error)
	CompareVersions(ctx context.Context, v1 string, v2 string) (result int, err error)
	GetAllManifests(ctx context.Context) (manifests []ManifestWithSource, err error)
}

// ManifestWithSource содержит манифест с информацией об источнике.
type ManifestWithSource struct {
	Path     string
	Source   string
	Manifest *models.Manifest
}

// PackageWithSource содержит пакет с информацией об источнике и манифесте.
type PackageWithSource struct {
	Source   string
	Package  *models.Package
	Manifest *models.Manifest
}

// ManifestInfo содержит информацию о манифесте в каталоге.
type ManifestInfo struct {
	URL      string
	Version  string
	LoadedAt string
}

// DependencyResolver разрешает зависимости.
type DependencyResolver interface {
	ResolveDependencies(ctx context.Context, pkg *models.Package) (graph *models.DependencyGraph, err error)
	CheckCycles(ctx context.Context, graph *models.DependencyGraph) (err error)
	SortForInstallation(ctx context.Context, graph *models.DependencyGraph) (packages []*models.Package, err error)
	CheckCompatibility(ctx context.Context, installed *models.Package, required *models.Dependency) (compatible bool)
}

// DownloadManager управляет загрузкой файлов.
type DownloadManager interface {
	Download(ctx context.Context, url string, destination string) (err error)
	DownloadWithProgress(ctx context.Context, url string, destination string, progress chan<- int) (err error)
}

// ValidationEngine выполняет валидацию файлов.
type ValidationEngine interface {
	ValidateChecksum(ctx context.Context, filePath string, algorithm string, expected string) (err error)
	ValidateSignature(ctx context.Context, filePath string, signature string) (err error)
	ValidateExecutable(ctx context.Context, filePath string) (err error)
	ValidateArchive(ctx context.Context, filePath string, format string) (err error)
}

// InstallationManager управляет установкой пакетов.
type InstallationManager interface {
	Install(ctx context.Context, pkg *models.Package, v models.Version) (err error)
	Uninstall(ctx context.Context, packageID string, keepFiles bool) (err error)
}

type ScopeManager interface {
	DeleteScope(ctx context.Context, name string, force bool) (err error)
	ListScopes(ctx context.Context) (scopes []models.ScopeInfo, err error)
	GetCurrentScope(ctx context.Context) (name string, err error)
	GetScopeConfig(ctx context.Context, scopeName string) (config *models.ScopeConfig, err error)
	CheckConsistency(ctx context.Context, scopeName string) (err error)
}

// DatabaseManager управляет базой данных установок.
type DatabaseManager interface {
	RecordInstallation(ctx context.Context, installation *models.Installation) (err error)
	RemoveInstallation(ctx context.Context, packageID string) (err error)
	GetInstallation(ctx context.Context, packageID string) (installation *models.Installation, err error)
	ListInstallations(ctx context.Context) (installations []models.Installation, err error)
	FindByPackage(ctx context.Context, source string, packageName string) (installation *models.Installation, err error)
}
