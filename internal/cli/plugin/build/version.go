// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"errors"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/mod/semver"
)

const defaultVersion = "0.0.1"

func resolveVersion(rootDir string) (version string) {

	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		return defaultVersion
	}

	tagRefs, err := repo.Tags()
	if err != nil {
		return defaultVersion
	}
	defer tagRefs.Close()

	var maxTag string
	for {
		var ref *plumbing.Reference
		if ref, err = tagRefs.Next(); errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return defaultVersion
		}
		name := ref.Name().Short()
		v := name
		if !strings.HasPrefix(v, "v") {
			v = "v" + v
		}
		if !semver.IsValid(v) {
			continue
		}
		if maxTag == "" || semver.Compare(v, maxTag) > 0 {
			maxTag = v
		}
	}

	if maxTag == "" {
		return defaultVersion
	}
	return strings.TrimPrefix(maxTag, "v")
}
