// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

// Встроенные интерфейсы оставляют невостребованные методы незаполненными: неожидаемый вызов паникует и проваливает тест.
type fakeDatabaseManager struct {
	managers.DatabaseManager
	installations []models.Installation
	listErr       error
}

func (f *fakeDatabaseManager) ListInstallations(ctx context.Context) (installations []models.Installation, err error) {

	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.installations, nil
}

type fakeManifestManager struct {
	managers.ManifestManager
	available    map[string][]managers.PackageWithSource
	fromSources  []models.Package
	findErr      error
	fromSourcesE error
	loadManifest *models.Manifest
	loadErr      error
}

func (f *fakeManifestManager) FindAllPackages(ctx context.Context, packageName string) (packages []managers.PackageWithSource, err error) {

	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.available[packageName], nil
}

func (f *fakeManifestManager) ListPackagesFromSources(ctx context.Context, sources map[string]bool) (packages []models.Package, err error) {

	if f.fromSourcesE != nil {
		return nil, f.fromSourcesE
	}
	return f.fromSources, nil
}

func (f *fakeManifestManager) LoadManifest(ctx context.Context, url string) (manifest *models.Manifest, err error) {

	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.loadManifest, nil
}

type fakeInstallationManager struct {
	installed      []string
	uninstalled    []string
	skillPackages  [][]string
	calls          []string
	installErr     error
	uninstallErr   error
	installSkillsE error
}

func (f *fakeInstallationManager) Install(ctx context.Context, pkg *models.Package, v models.Version) (err error) {

	f.calls = append(f.calls, "install")
	f.installed = append(f.installed, pkg.Name+"@"+v.Original)
	return f.installErr
}

func (f *fakeInstallationManager) Uninstall(ctx context.Context, packageID string, keepFiles bool) (err error) {

	f.calls = append(f.calls, "uninstall")
	f.uninstalled = append(f.uninstalled, packageID)
	return f.uninstallErr
}

func (f *fakeInstallationManager) InstallSkills(ctx context.Context, packageNames []string) (err error) {

	f.calls = append(f.calls, "skills")
	f.skillPackages = append(f.skillPackages, packageNames)
	return f.installSkillsE
}

func newTestInstaller(db *fakeDatabaseManager, manifest *fakeManifestManager, installation *fakeInstallationManager) (inst *Installer) {

	return &Installer{
		currentScope:        "test",
		databaseManager:     db,
		manifestManager:     manifest,
		installationManager: installation,
	}
}

func mustParse(t *testing.T, raw string) (parsed models.Version) {

	t.Helper()

	var err error
	if parsed, err = version.Parse(raw); err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return
}

func makeInstallation(id string, source string, name string, ver string) (result models.Installation) {

	return models.Installation{ID: id, Source: source, Package: name, Version: ver}
}

func TestKeepOnlyVersionRemovesOthers(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, installationMgr)

	group := []models.Installation{
		makeInstallation("id-6", "src", "astg", "1.0.6"),
		makeInstallation("id-7", "src", "astg", "1.0.7"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}

	removed, err := inst.keepOnlyVersion(context.Background(), group, mustParse(t, "1.0.8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 || removed[0] != "1.0.6" || removed[1] != "1.0.7" {
		t.Fatalf("removed=%v", removed)
	}
	if len(installationMgr.uninstalled) != 2 || installationMgr.uninstalled[0] != "id-6" || installationMgr.uninstalled[1] != "id-7" {
		t.Fatalf("uninstalled=%v", installationMgr.uninstalled)
	}
}

// Версия, записанная с префиксом v, семантически равна цели и не должна удаляться.
func TestKeepOnlyVersionMatchesSemantically(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, installationMgr)

	group := []models.Installation{
		makeInstallation("id-v8", "src", "astg", "v1.0.8"),
		makeInstallation("id-7", "src", "astg", "1.0.7"),
	}

	removed, err := inst.keepOnlyVersion(context.Background(), group, mustParse(t, "1.0.8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "1.0.7" {
		t.Fatalf("removed=%v", removed)
	}
}

func TestKeepOnlyVersionRemovesUnparsable(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, installationMgr)

	group := []models.Installation{
		makeInstallation("id-bad", "src", "astg", "not-a-version"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}

	removed, err := inst.keepOnlyVersion(context.Background(), group, mustParse(t, "1.0.8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "not-a-version" {
		t.Fatalf("removed=%v", removed)
	}
}

func TestKeepOnlyVersionStopsOnUninstallError(t *testing.T) {

	installationMgr := &fakeInstallationManager{uninstallErr: errors.New("boom")}
	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, installationMgr)

	group := []models.Installation{
		makeInstallation("id-6", "src", "astg", "1.0.6"),
		makeInstallation("id-7", "src", "astg", "1.0.7"),
	}

	if _, err := inst.keepOnlyVersion(context.Background(), group, mustParse(t, "1.0.8")); err == nil {
		t.Fatal("expected error")
	}
	if len(installationMgr.uninstalled) != 1 {
		t.Fatalf("uninstalled=%v", installationMgr.uninstalled)
	}
}

func TestLatestVersion(t *testing.T) {

	group := []models.Installation{
		makeInstallation("id-7", "src", "astg", "1.0.7"),
		makeInstallation("id-bad", "src", "astg", "garbage"),
		makeInstallation("id-10", "src", "astg", "1.0.10"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}

	if latest := latestVersion(group); latest.Original != "1.0.10" {
		t.Fatalf("latest=%q", latest.Original)
	}
	if latest := latestVersion(nil); latest.Original != "" {
		t.Fatalf("latest=%q", latest.Original)
	}
	if latest := latestVersion([]models.Installation{makeInstallation("id", "src", "astg", "garbage")}); latest.Original != "" {
		t.Fatalf("latest=%q", latest.Original)
	}
}

// Один и тот же пакет из разных источников — независимые группы.
func TestGroupInstallationsSplitsBySource(t *testing.T) {

	groups := groupInstallations([]models.Installation{
		makeInstallation("a", "src-1", "astg", "1.0.7"),
		makeInstallation("b", "src-1", "astg", "1.0.8"),
		makeInstallation("c", "src-2", "astg", "1.0.8"),
	})

	if len(groups) != 2 {
		t.Fatalf("groups=%d", len(groups))
	}
	if got := len(groups[packageKey{source: "src-1", name: "astg"}]); got != 2 {
		t.Fatalf("src-1 group=%d", got)
	}
	if got := len(groups[packageKey{source: "src-2", name: "astg"}]); got != 1 {
		t.Fatalf("src-2 group=%d", got)
	}
}

func TestSelectGroups(t *testing.T) {

	groups := groupInstallations([]models.Installation{
		makeInstallation("a", "src-2", "server", "1.0.8"),
		makeInstallation("b", "src-1", "astg", "1.0.8"),
		makeInstallation("c", "src-2", "astg", "1.0.8"),
	})

	all := selectGroups(groups, nil)
	if len(all) != 3 {
		t.Fatalf("all=%+v", all)
	}
	if all[0].name != "astg" || all[0].source != "src-1" || all[1].source != "src-2" || all[2].name != "server" {
		t.Fatalf("order=%+v", all)
	}

	filtered := selectGroups(groups, []string{"server"})
	if len(filtered) != 1 || filtered[0].name != "server" {
		t.Fatalf("filtered=%+v", filtered)
	}

	if unknown := selectGroups(groups, []string{"missing"}); len(unknown) != 0 {
		t.Fatalf("unknown=%+v", unknown)
	}
}
