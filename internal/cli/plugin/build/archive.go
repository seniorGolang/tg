// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	skillsDirName    = "skills"
	skillFileName    = "SKILL.md"
	skillsArchiveSfx = "-skills.tar.gz"
)

type skillEntry struct {
	Name  string
	Files []string // paths relative to skill root (incl. SKILL.md)
}

func collectSkills(pluginDir string) (entries []skillEntry, err error) {

	skillsRoot := filepath.Join(pluginDir, skillsDirName)
	var dirs []os.DirEntry
	if dirs, err = os.ReadDir(skillsRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	entries = make([]skillEntry, 0)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsRoot, dir.Name())
		if _, statErr := os.Stat(filepath.Join(skillPath, skillFileName)); statErr != nil {
			continue
		}

		entry := skillEntry{Name: dir.Name(), Files: make([]string, 0)}
		if err = filepath.WalkDir(skillPath, func(path string, d fs.DirEntry, walkErr error) (err error) {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			var rel string
			if rel, err = filepath.Rel(skillPath, path); err != nil {
				return err
			}
			entry.Files = append(entry.Files, filepath.ToSlash(rel))
			return nil
		}); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return
}

func writeSkillsArchive(pluginDir string, archivePath string, entries []skillEntry) (err error) {

	var out *os.File
	if out, err = os.Create(archivePath); err != nil {
		return
	}
	defer func() {
		closeErr := out.Close()
		if err == nil {
			err = closeErr
		}
	}()

	gz := gzip.NewWriter(out)
	defer func() {
		closeErr := gz.Close()
		if err == nil {
			err = closeErr
		}
	}()

	tw := tar.NewWriter(gz)
	defer func() {
		closeErr := tw.Close()
		if err == nil {
			err = closeErr
		}
	}()

	skillsRoot := filepath.Join(pluginDir, skillsDirName)
	for _, entry := range entries {
		for _, rel := range entry.Files {
			full := filepath.Join(skillsRoot, entry.Name, filepath.FromSlash(rel))
			var info os.FileInfo
			if info, err = os.Stat(full); err != nil {
				return
			}
			headerName := entry.Name + "/" + rel
			var header *tar.Header
			if header, err = tar.FileInfoHeader(info, ""); err != nil {
				return
			}
			header.Name = headerName
			if err = tw.WriteHeader(header); err != nil {
				return
			}
			var file *os.File
			if file, err = os.Open(full); err != nil {
				return
			}
			if _, err = io.Copy(tw, file); err != nil {
				_ = file.Close()
				return
			}
			if err = file.Close(); err != nil {
				return
			}
		}
	}
	return
}

func packagePluginSkills(rootDir string, outDir string, b *builtPlugin) (err error) {

	pluginDir := filepath.Join(rootDir, "plugins", b.Dir)
	var entries []skillEntry
	if entries, err = collectSkills(pluginDir); err != nil {
		return
	}
	if len(entries) == 0 {
		return
	}

	archiveName := b.Name + skillsArchiveSfx
	archivePath := filepath.Join(outDir, archiveName)
	if err = writeSkillsArchive(pluginDir, archivePath, entries); err != nil {
		return fmt.Errorf("package skills for %s: %w", b.Name, err)
	}

	b.SkillsArchive = archiveName
	b.Skills = make([]builtSkill, 0, len(entries))
	for _, entry := range entries {
		b.Skills = append(b.Skills, builtSkill{
			Name:  entry.Name,
			Files: entry.Files,
			Root:  skillsDirName + "/" + b.Dir + "/" + entry.Name,
		})
	}
	return
}
