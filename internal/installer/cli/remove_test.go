// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"reflect"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// Удаление одной из сосуществующих версий стирает общий published путь, поэтому skills переустанавливаются после снятия.
func TestHandleRemoveRepublishesSkills(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-7", "src", "astg", "1.0.7"),
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandleRemove(context.Background(), []string{"astg"}, true, false); err != nil {
		t.Fatal(err)
	}

	if want := []string{"uninstall", "skills"}; !reflect.DeepEqual(installationMgr.calls, want) {
		t.Fatalf("calls=%v want=%v", installationMgr.calls, want)
	}
	if want := [][]string{{"astg"}}; !reflect.DeepEqual(installationMgr.skillPackages, want) {
		t.Fatalf("skillPackages=%v want=%v", installationMgr.skillPackages, want)
	}
}

func TestHandleRemoveNotInstalled(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, installationMgr)

	if err := inst.HandleRemove(context.Background(), []string{"astg"}, true, false); err == nil {
		t.Fatal("expected error")
	}
	if len(installationMgr.calls) != 0 {
		t.Fatalf("calls=%v", installationMgr.calls)
	}
}

// dry-run не должен ничего менять.
func TestHandleRemoveDryRun(t *testing.T) {

	db := &fakeDatabaseManager{installations: []models.Installation{
		makeInstallation("id-8", "src", "astg", "1.0.8"),
	}}
	installationMgr := &fakeInstallationManager{}

	inst := newTestInstaller(db, &fakeManifestManager{}, installationMgr)
	if err := inst.HandleRemove(context.Background(), []string{"astg"}, true, true); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.calls) != 0 {
		t.Fatalf("calls=%v", installationMgr.calls)
	}
}
