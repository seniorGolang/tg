// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/storage"
	"github.com/seniorGolang/tg/v3/internal/installer/uri"
	"github.com/seniorGolang/tg/v3/internal/installer/version"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

// HandleUpdate обрабатывает команду update.
func (inst *Installer) HandleUpdate(ctx context.Context, args []string, force bool) (err error) {

	if len(args) > 0 {
		source := args[0]
		pterm.Info.Printf(i18n.Msg("Updating manifest: %s")+"\n", source)

		// Нормализуем source через URI (аналогично TransformURL)
		var parsedURI uri.URI
		if parsedURI, err = uri.New(source); err == nil {
			source = parsedURI.Source()
		}

		// Получаем manifestURL через parsedURI
		if parsedURI, err = uri.New(source); err != nil {
			err = fmt.Errorf("failed to parse source URL: %w", err)
			return
		}

		var manifestURL string
		if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to get manifest for %s: %w"), source, err)
			return
		}

		if err = inst.updateManifestWithURL(ctx, source, manifestURL, force); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to update manifest %s: %w"), source, err)
			return
		}
		inst.printPackageChanges(ctx, source)
		pterm.Success.Printf(i18n.Msg("Manifest updated successfully: %s")+"\n", source)
		return
	}

	var catalog []managers.ManifestInfo
	if catalog, err = inst.manifestManager.GetCatalog(ctx); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get catalog: %w"), err)
		return
	}

	if len(catalog) == 0 {
		pterm.Info.Println(i18n.Msg("No manifests found in catalog"))
		return
	}

	totalManifests := len(catalog)
	if totalManifests > 1 {
		pterm.Info.Printf(i18n.Msg("Updating %d manifests...")+"\n", totalManifests)
	}

	for i, info := range catalog {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		if totalManifests > 1 {
			pterm.Info.Printf(i18n.Msg("Updating manifest %d/%d: %s")+"\n", i+1, totalManifests, info.URL)
		} else {
			pterm.Info.Printf(i18n.Msg("Updating manifest: %s")+"\n", info.URL)
		}

		// Нормализуем source через URI (аналогично TransformURL)
		source := info.URL
		var parsedURI uri.URI
		if parsedURI, err = uri.New(source); err == nil {
			source = parsedURI.Source()
		}

		// Получаем manifestURL через parsedURI
		if parsedURI, err = uri.New(source); err != nil {
			err = fmt.Errorf("failed to parse source URL: %w", err)
			return
		}

		var manifestURL string
		if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to get manifest for %s: %w"), info.URL, err)
			return
		}

		if err = inst.updateManifestWithURL(ctx, info.URL, manifestURL, force); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to update manifest %s: %w"), info.URL, err)
			return
		}

		inst.printPackageChanges(ctx, info.URL)
		pterm.Success.Printf(i18n.Msg("Manifest updated successfully: %s")+"\n", info.URL)
	}

	if totalManifests > 1 {
		pterm.Success.Printf(i18n.Msg("Successfully updated %d manifest(s)")+"\n", totalManifests)
	}

	return
}

// updateManifestWithURL обновляет манифест используя уже построенный manifestURL.
func (inst *Installer) updateManifestWithURL(ctx context.Context, source string, manifestURL string, force bool) (err error) {

	normalizedSource := storage.NormalizeSource(source)
	manifestDir := storage.GetManifestDir(inst.currentScope, normalizedSource)

	// Если force=false, проверяем версию и пропускаем, если новая версия меньше
	if !force {
		var existingVersion string
		if existingVersion, err = inst.getExistingManifestVersion(manifestDir); err == nil {
			var newManifest *models.Manifest
			if newManifest, err = inst.manifestManager.LoadManifest(ctx, manifestURL); err != nil {
				err = fmt.Errorf(i18n.Msg("Failed to load manifest: %w"), err)
				return
			}

			var newVersion models.Version
			if newVersion, err = version.Parse(newManifest.Version); err != nil {
				err = fmt.Errorf(i18n.Msg("Invalid new manifest version format: %w"), err)
				return
			}

			var oldVersion models.Version
			if oldVersion, err = version.Parse(existingVersion); err == nil {
				comparison := version.Compare(newVersion, oldVersion)
				if comparison < 0 {
					return
				}
			}
		}
	}

	_, err = inst.manifestManager.LoadManifestCascade(ctx, manifestURL, source, force)
	return
}

func (inst *Installer) getExistingManifestVersion(manifestDir string) (version string, err error) {

	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)
	var statErr error
	if _, statErr = os.Stat(manifestFile); os.IsNotExist(statErr) {
		err = errors.New(i18n.Msg("Manifest file not found"))
		return
	}

	var data []byte
	if data, err = os.ReadFile(manifestFile); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to read manifest file: %w"), err)
		return
	}

	var manifest models.Manifest
	if err = yaml.Unmarshal(data, &manifest); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to parse manifest file: %w"), err)
		return
	}

	version = manifest.Version
	return
}

// printPackageChanges выводит изменения в пакетах после обновления манифеста.
func (inst *Installer) printPackageChanges(ctx context.Context, source string) {

	var parsedURI uri.URI
	var err error
	if parsedURI, err = uri.New(source); err != nil {
		return
	}

	var manifestURL string
	if manifestURL, err = parsedURI.ManifestURL(ctx, ""); err != nil {
		return
	}

	var newManifest *models.Manifest
	if newManifest, err = inst.manifestManager.LoadManifest(ctx, manifestURL); err != nil {
		return
	}

	// Читаем старый манифест для сравнения
	normalizedSource := storage.NormalizeSource(source)
	manifestDir := storage.GetManifestDir(inst.currentScope, normalizedSource)
	manifestFile := filepath.Join(manifestDir, storage.ManifestFileName)

	var oldManifest *models.Manifest
	var statErr error
	if _, statErr = os.Stat(manifestFile); statErr == nil {
		var data []byte
		var readErr error
		if data, readErr = os.ReadFile(manifestFile); readErr == nil {
			var mf models.Manifest
			var unmarshalErr error
			if unmarshalErr = yaml.Unmarshal(data, &mf); unmarshalErr == nil {
				oldManifest = &mf
			}
		}
	}

	// Создаем карту пакетов из старого манифеста
	oldPackagesMap := make(map[string]bool)
	if oldManifest != nil {
		for _, pkg := range oldManifest.Packages {
			oldPackagesMap[pkg.Name] = true
		}
	}

	var installations []models.Installation
	if installations, err = inst.databaseManager.ListInstallations(ctx); err != nil {
		return
	}

	// Создаем карту установленных пакетов по source + package name
	// Для каждого пакета берем запись с максимальной версией
	installedMap := make(map[string]*models.Installation)
	for i := range installations {
		if installations[i].Source == source {
			key := installations[i].Package
			existing, exists := installedMap[key]
			if !exists {
				installedMap[key] = &installations[i]
			} else {
				// Сравниваем версии и берем более новую
				existingVersion, err1 := version.Parse(existing.Version)
				newVersion, err2 := version.Parse(installations[i].Version)
				if err1 == nil && err2 == nil {
					comparison := version.Compare(newVersion, existingVersion)
					if comparison > 0 {
						installedMap[key] = &installations[i]
					}
				}
			}
		}
	}

	type packageChange struct {
		name        string
		oldVersion  string
		newVersion  string
		changeType  string
		isInstalled bool
	}

	var updated, newPackages, unchanged []packageChange
	var newManifestVersion models.Version
	if newManifestVersion, err = version.Parse(newManifest.Version); err != nil {
		return
	}

	var oldManifestVersion models.Version
	if oldManifest != nil {
		oldManifestVersion, err = version.Parse(oldManifest.Version)
		if err != nil {
			oldManifestVersion = models.Version{}
		}
	}

	for _, pkg := range newManifest.Packages {
		// Проверяем, был ли пакет в старом манифесте
		wasInOldManifest := oldPackagesMap[pkg.Name]

		installed, isInstalled := installedMap[pkg.Name]

		// Новый пакет - тот, которого не было в старом манифесте
		if !wasInOldManifest {
			newPackages = append(newPackages, packageChange{
				name:        pkg.Name,
				newVersion:  newManifest.Version,
				changeType:  changeTypeNew,
				isInstalled: isInstalled,
			})
			continue
		}

		// Если пакет был в старом манифесте, проверяем, обновился ли он
		if !isInstalled {
			// Пакет был в старом манифесте, но не установлен
			// Сравниваем версии манифестов
			if oldManifestVersion.Original != "" {
				comparison := version.Compare(newManifestVersion, oldManifestVersion)
				if comparison > 0 {
					updated = append(updated, packageChange{
						name:        pkg.Name,
						oldVersion:  oldManifest.Version,
						newVersion:  newManifest.Version,
						changeType:  changeTypeUpdated,
						isInstalled: false,
					})
				} else if comparison == 0 {
					unchanged = append(unchanged, packageChange{
						name:        pkg.Name,
						oldVersion:  oldManifest.Version,
						newVersion:  newManifest.Version,
						changeType:  changeTypeUnchanged,
						isInstalled: false,
					})
				}
			}
			continue
		}

		// Пакет установлен - сравниваем версию манифеста, из которого был установлен пакет, с версией нового манифеста
		var installedVersion models.Version
		if installedVersion, err = version.Parse(installed.Version); err != nil {
			continue
		}

		comparison := version.Compare(newManifestVersion, installedVersion)
		if comparison > 0 {
			updated = append(updated, packageChange{
				name:        pkg.Name,
				oldVersion:  installed.Version,
				newVersion:  newManifest.Version,
				changeType:  changeTypeUpdated,
				isInstalled: true,
			})
		} else if comparison == 0 {
			unchanged = append(unchanged, packageChange{
				name:        pkg.Name,
				oldVersion:  installed.Version,
				newVersion:  newManifest.Version,
				changeType:  changeTypeUnchanged,
				isInstalled: true,
			})
		}
	}

	if len(updated) > 0 || len(newPackages) > 0 || len(unchanged) > 0 {
		// Строим дерево для pterm
		var categoryNodes []pterm.TreeNode

		// Узел "Updated packages"
		if len(updated) > 0 {
			var updatedNodes []pterm.TreeNode
			for _, pkg := range updated {
				versionText := fmt.Sprintf("%s (%s -> %s)", pkg.name, pkg.oldVersion, pkg.newVersion)
				if pkg.isInstalled {
					versionText += " " + pterm.Green("✓")
				}
				updatedNodes = append(updatedNodes, pterm.TreeNode{
					Text: versionText,
				})
			}
			categoryNodes = append(categoryNodes, pterm.TreeNode{
				Text:     fmt.Sprintf(i18n.Msg("Updated packages (%d)"), len(updated)),
				Children: updatedNodes,
			})
		}

		// Узел "New packages"
		if len(newPackages) > 0 {
			var newNodes []pterm.TreeNode
			for _, pkg := range newPackages {
				packageText := pkg.name
				if pkg.isInstalled {
					packageText += " " + pterm.Green("✓")
				}
				newNodes = append(newNodes, pterm.TreeNode{
					Text: packageText,
				})
			}
			categoryNodes = append(categoryNodes, pterm.TreeNode{
				Text:     fmt.Sprintf(i18n.Msg("New packages (%d)"), len(newPackages)),
				Children: newNodes,
			})
		}

		// Узел "Unchanged packages"
		if len(unchanged) > 0 {
			var unchangedNodes []pterm.TreeNode
			for _, pkg := range unchanged {
				versionText := pkg.name
				if pkg.newVersion != "" {
					versionText = fmt.Sprintf("%s (%s)", pkg.name, pkg.newVersion)
				}
				if pkg.isInstalled {
					versionText += " " + pterm.Green("✓")
				}
				unchangedNodes = append(unchangedNodes, pterm.TreeNode{
					Text: versionText,
				})
			}
			categoryNodes = append(categoryNodes, pterm.TreeNode{
				Text:     fmt.Sprintf(i18n.Msg("Unchanged packages (%d)"), len(unchanged)),
				Children: unchangedNodes,
			})
		}

		// Корневой узел - source
		rootNode := pterm.TreeNode{
			Text:     source,
			Children: categoryNodes,
		}

		// Выводим дерево через pterm
		pterm.Println()
		if err = pterm.DefaultTree.WithRoot(rootNode).Render(); err != nil {
			return
		}
		pterm.Println()
	}
}
