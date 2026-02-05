// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"errors"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const defaultVersion = "0.0.1-dev"

func resolveVersion(rootDir string) (version string) {

	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		return defaultVersion
	}

	head, err := repo.Head()
	if err != nil {
		return defaultVersion
	}

	tagByCommit, err := tagToCommitMap(repo)
	if err != nil {
		return defaultVersion
	}

	if tagName, ok := tagByCommit[head.Hash()]; ok {
		return stripVPrefix(tagName)
	}

	cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return defaultVersion
	}
	defer cIter.Close()

	for {
		var commit *object.Commit
		if commit, err = cIter.Next(); errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return defaultVersion
		}
		if tagName, ok := tagByCommit[commit.Hash]; ok {
			return stripVPrefix(tagName)
		}
	}

	return defaultVersion
}

func tagToCommitMap(repo *git.Repository) (out map[plumbing.Hash]string, err error) {

	out = make(map[plumbing.Hash]string)
	tagRefs, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	defer tagRefs.Close()

	for {
		var ref *plumbing.Reference
		if ref, err = tagRefs.Next(); errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		var commitHash plumbing.Hash
		if commitHash, err = resolveTagToCommit(repo, ref); err != nil {
			continue
		}
		out[commitHash] = ref.Name().Short()
	}

	return
}

func resolveTagToCommit(repo *git.Repository, ref *plumbing.Reference) (hash plumbing.Hash, err error) {

	h := ref.Hash()
	for {
		var obj object.Object
		if obj, err = repo.Object(plumbing.AnyObject, h); err != nil {
			hash = plumbing.ZeroHash
			return
		}
		switch o := obj.(type) {
		case *object.Commit:
			hash = o.Hash
			return
		case *object.Tag:
			if o.TargetType != plumbing.CommitObject && o.TargetType != plumbing.TagObject {
				hash = plumbing.ZeroHash
				err = errors.New("tag targets non-commit")
				return
			}
			h = o.Target
		default:
			hash = plumbing.ZeroHash
			err = errors.New("unsupported tag target type")
			return
		}
	}
}

func stripVPrefix(tag string) (version string) {

	version = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if version == "" {
		return defaultVersion
	}
	return version
}
