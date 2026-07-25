// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"testing"

	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

// Явная версия избавляет getPackageVersion от обращения к манифесту пакета.
const allExplicitVersion = "1.0.0"

// --all устанавливает все пакеты источника, минуя интерактивный выбор.
func TestProcessAllPackagesInstallsEverything(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	manifestMgr := &fakeManifestManager{
		fromSources: []models.Package{{Name: "astg"}, {Name: "server"}},
	}
	inst := newTestInstaller(&fakeDatabaseManager{}, manifestMgr, installationMgr)

	if err := inst.processAllPackagesFromManifests(context.Background(), allExplicitVersion, false, "file:///tmp/dist", nil, true); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.installed) != 2 {
		t.Fatalf("installed=%v", installationMgr.installed)
	}
	if installationMgr.installed[0] != "astg@1.0.0" || installationMgr.installed[1] != "server@1.0.0" {
		t.Fatalf("installed=%v", installationMgr.installed)
	}
}

// Один пакет обрабатывается быстрым путём независимо от флага all.
func TestProcessAllPackagesSinglePackage(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	manifestMgr := &fakeManifestManager{
		fromSources: []models.Package{{Name: "astg"}},
	}
	inst := newTestInstaller(&fakeDatabaseManager{}, manifestMgr, installationMgr)

	if err := inst.processAllPackagesFromManifests(context.Background(), allExplicitVersion, false, "file:///tmp/dist", nil, true); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.installed) != 1 || installationMgr.installed[0] != "astg@1.0.0" {
		t.Fatalf("installed=%v", installationMgr.installed)
	}
}

// Пустой источник не приводит к установкам и ошибкам.
func TestProcessAllPackagesEmpty(t *testing.T) {

	installationMgr := &fakeInstallationManager{}
	inst := newTestInstaller(&fakeDatabaseManager{}, &fakeManifestManager{}, installationMgr)

	if err := inst.processAllPackagesFromManifests(context.Background(), allExplicitVersion, false, "file:///tmp/dist", nil, true); err != nil {
		t.Fatal(err)
	}

	if len(installationMgr.installed) != 0 {
		t.Fatalf("installed=%v", installationMgr.installed)
	}
}
