// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package skills

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Publish копирует дерево skill из sourceDir в destDir (replace).
func Publish(sourceDir string, destDir string) (err error) {

	var info os.FileInfo
	if info, err = os.Stat(sourceDir); err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("skill source is not a directory: %s", sourceDir)
	}

	if _, err = os.Stat(destDir); err == nil {
		if err = os.RemoveAll(destDir); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return
	}

	return copyTree(sourceDir, destDir)
}

// Remove удаляет опубликованный каталог skill, если он существует.
func Remove(destDir string) (err error) {

	if _, err = os.Stat(destDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(destDir)
}

func copyTree(src string, dst string) (err error) {

	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) (err error) {

		if walkErr != nil {
			return walkErr
		}

		var rel string
		if rel, err = filepath.Rel(src, path); err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		return copyFile(path, target)
	})
}

func copyFile(src string, dst string) (err error) {

	var in *os.File
	if in, err = os.Open(src); err != nil {
		return
	}
	defer func() { _ = in.Close() }()

	var out *os.File
	if out, err = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer func() {
		closeErr := out.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return
	}
	return
}
