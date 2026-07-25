// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	"github.com/seniorGolang/tg/v3/internal/installer/version"
)

// packageKey идентифицирует набор установок одного пакета из одного источника.
type packageKey struct {
	source string
	name   string
}

// groupInstallations группирует установки по источнику и имени пакета.
func groupInstallations(installations []models.Installation) (groups map[packageKey][]models.Installation) {

	groups = make(map[packageKey][]models.Installation, len(installations))
	for _, record := range installations {
		key := packageKey{source: record.Source, name: record.Package}
		groups[key] = append(groups[key], record)
	}

	return
}

// selectGroups отбирает группы по именам пакетов (все группы при пустом names) в детерминированном порядке.
func selectGroups(groups map[packageKey][]models.Installation, names []string) (keys []packageKey) {

	filter := make(map[string]struct{}, len(names))
	for _, name := range names {
		filter[name] = struct{}{}
	}

	keys = make([]packageKey, 0, len(groups))
	for key := range groups {
		if len(filter) > 0 {
			if _, ok := filter[key.name]; !ok {
				continue
			}
		}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) (less bool) {

		if keys[i].name != keys[j].name {
			return keys[i].name < keys[j].name
		}
		return keys[i].source < keys[j].source
	})

	return
}

// latestVersion возвращает наибольшую версию среди установок группы; непарсящиеся версии игнорируются.
func latestVersion(group []models.Installation) (latest models.Version) {

	for i := range group {
		var parsed models.Version
		var parseErr error
		if parsed, parseErr = version.Parse(group[i].Version); parseErr != nil {
			continue
		}
		if latest.Original == "" || version.Compare(parsed, latest) > 0 {
			latest = parsed
		}
	}

	return
}

// keepOnlyVersion удаляет все установки группы, кроме keep, и возвращает удалённые версии.
func (inst *Installer) keepOnlyVersion(ctx context.Context, group []models.Installation, keep models.Version) (removed []string, err error) {

	removed = make([]string, 0, len(group))
	for i := range group {
		if group[i].Version == keep.Original {
			continue
		}
		var parsed models.Version
		var parseErr error
		if parsed, parseErr = version.Parse(group[i].Version); parseErr == nil && version.Compare(parsed, keep) == 0 {
			continue
		}
		if err = inst.installationManager.Uninstall(ctx, group[i].ID, false); err != nil {
			return removed, fmt.Errorf(i18n.Msg("Failed to remove package %s@%s: %w"), group[i].Package, group[i].Version, err)
		}
		removed = append(removed, group[i].Version)
	}

	return
}
