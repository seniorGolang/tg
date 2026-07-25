// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
)

const catalogSource = "file:///tmp/tgp-go/dist"

// installedSource — источник в том виде, в котором его записывает установщик.
func installedSource() (source string) {

	return storage.BaseSourceURL(catalogSource)
}

func availablePackage(name string, manifestVersion string) (pkg managers.PackageWithSource) {

	return managers.PackageWithSource{
		Source:   catalogSource,
		Package:  &models.Package{Name: name},
		Manifest: &models.Manifest{Version: manifestVersion},
	}
}

func TestHandleUpgradeReplacesOldVersions(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", installedSource(), "astg", "1.0.7"),
		makeInstallation("id-6", installedSource(), "astg", "1.0.6"),
	}}
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg": {availablePackage("astg", "1.0.7"), availablePackage("astg", "1.0.8")},
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"astg@1.0.8"}; !reflect.DeepEqual(installationMgr.installed, want) {
		t.Fatalf("installed=%v want=%v", installationMgr.installed, want)
	}
	if want := []string{"id-7", "id-6"}; !reflect.DeepEqual(installationMgr.uninstalled, want) {
		t.Fatalf("uninstalled=%v want=%v", installationMgr.uninstalled, want)
	}
	if want := [][]string{{"astg"}}; !reflect.DeepEqual(installationMgr.skillPackages, want) {
		t.Fatalf("skillPackages=%v want=%v", installationMgr.skillPackages, want)
	}
}

// Порядок критичен: published skills адресуются по имени, поэтому переустановка обязана идти после снятия старых версий.
func TestHandleUpgradeRepublishesSkillsLast(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", installedSource(), "astg", "1.0.7"),
	}}
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg": {availablePackage("astg", "1.0.8")},
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"install", "uninstall", "skills"}; !reflect.DeepEqual(installationMgr.calls, want) {
		t.Fatalf("calls=%v want=%v", installationMgr.calls, want)
	}
}

func TestHandleUpgradeAlreadyLatest(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-8", installedSource(), "astg", "1.0.8"),
	}}
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg": {availablePackage("astg", "1.0.8")},
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.uninstalled) != 0 {
		t.Fatalf("uninstalled=%v", installationMgr.uninstalled)
	}
	if want := []string{"astg@1.0.8"}; !reflect.DeepEqual(installationMgr.installed, want) {
		t.Fatalf("installed=%v want=%v", installationMgr.installed, want)
	}
}

func TestHandleUpgradeFiltersByName(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("astg-7", installedSource(), "astg", "1.0.7"),
		makeInstallation("server-7", installedSource(), "server", "1.0.7"),
	}}
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg":   {availablePackage("astg", "1.0.8")},
		"server": {availablePackage("server", "1.0.8")},
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), []string{"server"}); err != nil {
		t.Fatal(err)
	}

	if want := []string{"server@1.0.8"}; !reflect.DeepEqual(installationMgr.installed, want) {
		t.Fatalf("installed=%v want=%v", installationMgr.installed, want)
	}
	if want := []string{"server-7"}; !reflect.DeepEqual(installationMgr.uninstalled, want) {
		t.Fatalf("uninstalled=%v want=%v", installationMgr.uninstalled, want)
	}
}

func TestHandleUpgradeUnknownPackage(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", installedSource(), "astg", "1.0.7"),
	}}
	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)

	if err := inst.HandleUpgrade(context.Background(), []string{"missing"}); err == nil {
		t.Fatal("expected error")
	}
	if len(installationMgr.calls) != 0 {
		t.Fatalf("calls=%v", installationMgr.calls)
	}
}

func TestHandleUpgradeNoInstallations(t *testing.T) {

	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, &fakeInstallationManager{})

	if err := inst.HandleUpgrade(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

// Пакет установлен, но отсутствует в каталоге источника: обновлять нечем.
func TestHandleUpgradeNotInCatalog(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", installedSource(), "astg", "1.0.7"),
	}}
	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)

	if err := inst.HandleUpgrade(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
	if len(installationMgr.installed) != 0 || len(installationMgr.uninstalled) != 0 {
		t.Fatalf("installed=%v uninstalled=%v", installationMgr.installed, installationMgr.uninstalled)
	}
}

// Версия того же пакета из чужого источника не должна выбираться как цель обновления.
func TestHandleUpgradeIgnoresForeignSource(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", installedSource(), "astg", "1.0.7"),
	}}
	foreign := availablePackage("astg", "2.0.0")
	foreign.Source = "file:///tmp/other/dist"
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg": {foreign, availablePackage("astg", "1.0.8")},
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"astg@1.0.8"}; !reflect.DeepEqual(installationMgr.installed, want) {
		t.Fatalf("installed=%v want=%v", installationMgr.installed, want)
	}
}

// Установка новой версии провалилась — старые версии обязаны остаться на месте.
func TestHandleUpgradeKeepsOldVersionsOnInstallError(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", installedSource(), "astg", "1.0.7"),
	}}
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg": {availablePackage("astg", "1.0.8")},
	}}
	installationMgr := &fakeInstallationManager{installErr: errors.New("download failed")}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
	if len(installationMgr.uninstalled) != 0 {
		t.Fatalf("uninstalled=%v", installationMgr.uninstalled)
	}
}

func TestHandleUpgradeListError(t *testing.T) {

	db := &fakeDatabaseManager{listErr: errors.New("db down")}
	inst := newTestInstaller(db, &fakeManifestManager{}, &fakeInstallationManager{})

	if err := inst.HandleUpgrade(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

// Каждый источник схлопывается независимо.
func TestHandleUpgradeHandlesEachSource(t *testing.T) {

	otherCatalog := "file:///tmp/other/dist"
	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("a-7", installedSource(), "astg", "1.0.7"),
		makeInstallation("b-7", storage.BaseSourceURL(otherCatalog), "astg", "1.0.7"),
	}}
	other := availablePackage("astg", "1.0.9")
	other.Source = otherCatalog
	manifest := &fakeManifestManager{available: map[string][]managers.PackageWithSource{
		"astg": {availablePackage("astg", "1.0.8"), other},
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, manifest, installationMgr)
	if err := inst.HandleUpgrade(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"astg@1.0.9", "astg@1.0.8"}; !reflect.DeepEqual(installationMgr.installed, want) {
		t.Fatalf("installed=%v want=%v", installationMgr.installed, want)
	}
	// Группы обходятся в детерминированном порядке: имя, затем источник.
	if want := []string{"b-7", "a-7"}; !reflect.DeepEqual(installationMgr.uninstalled, want) {
		t.Fatalf("uninstalled=%v want=%v", installationMgr.uninstalled, want)
	}
}
