// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func TestHandlePruneKeepsLatest(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", "src", "astg", "1.0.7"),
		makeInstallation("id-10", "src", "astg", "1.0.10"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"id-7", "id-8"}; !reflect.DeepEqual(installationMgr.uninstalled, want) {
		t.Fatalf("uninstalled=%v want=%v", installationMgr.uninstalled, want)
	}
	if len(installationMgr.installed) != 0 {
		t.Fatalf("prune must not install: %v", installationMgr.installed)
	}
	if want := [][]string{{"astg"}}; !reflect.DeepEqual(installationMgr.skillPackages, want) {
		t.Fatalf("skillPackages=%v want=%v", installationMgr.skillPackages, want)
	}
}

// Снятие старых версий стирает published путь, поэтому skills выжившей версии переустанавливаются после удаления.
func TestHandlePruneRepublishesSkillsLast(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", "src", "astg", "1.0.7"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"uninstall", "skills"}; !reflect.DeepEqual(installationMgr.calls, want) {
		t.Fatalf("calls=%v want=%v", installationMgr.calls, want)
	}
}

func TestHandlePruneSingleVersionIsNoop(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.calls) != 0 {
		t.Fatalf("calls=%v", installationMgr.calls)
	}
}

// Каждый источник обрабатывается отдельно: версия из чужого источника не считается лишней.
func TestHandlePruneIsolatesSources(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("a-7", "src-1", "astg", "1.0.7"),
		makeInstallation("a-8", "src-1", "astg", "1.0.8"),
		makeInstallation("b-7", "src-2", "astg", "1.0.7"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if want := []string{"a-7"}; !reflect.DeepEqual(installationMgr.uninstalled, want) {
		t.Fatalf("uninstalled=%v want=%v", installationMgr.uninstalled, want)
	}
}

func TestHandlePruneFiltersByName(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("astg-7", "src", "astg", "1.0.7"),
		makeInstallation("astg-8", "src", "astg", "1.0.8"),
		makeInstallation("server-7", "src", "server", "1.0.7"),
		makeInstallation("server-8", "src", "server", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), []string{"server"}); err != nil {
		t.Fatal(err)
	}

	if want := []string{"server-7"}; !reflect.DeepEqual(installationMgr.uninstalled, want) {
		t.Fatalf("uninstalled=%v want=%v", installationMgr.uninstalled, want)
	}
}

// Все версии группы непарсящиеся: выбрать выжившую нельзя, поэтому группа не трогается.
func TestHandlePruneSkipsUnparsableGroup(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-a", "src", "astg", "garbage"),
		makeInstallation("id-b", "src", "astg", "also-bad"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.calls) != 0 {
		t.Fatalf("calls=%v", installationMgr.calls)
	}
}

func TestHandlePruneNoInstallations(t *testing.T) {

	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, &fakeInstallationManager{})

	if err := inst.HandlePrune(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestHandlePruneUnknownPackage(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	inst := newTestInstaller(db, &fakeManifestManager{}, &fakeInstallationManager{})

	if err := inst.HandlePrune(context.Background(), []string{"missing"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestHandlePruneListError(t *testing.T) {

	db := &fakeDatabaseManager{listErr: errors.New("db down")}
	inst := newTestInstaller(db, &fakeManifestManager{}, &fakeInstallationManager{})

	if err := inst.HandlePrune(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestHandlePruneUninstallError(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", "src", "astg", "1.0.7"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{uninstallErr: errors.New("locked")}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandlePrune(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
	if len(installationMgr.skillPackages) != 0 {
		t.Fatalf("skillPackages=%v", installationMgr.skillPackages)
	}
}
