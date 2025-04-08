package generator

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v2/pkg/mod"
)

//go:embed ts/*
var tsFiles embed.FS

//go:embed pkg/*
var pkgFiles embed.FS

func pkgCopyTo(pkg, dst string) (err error) {

	pkgPath := path.Join("pkg", pkg)
	var entries []fs.DirEntry
	if entries, err = pkgFiles.ReadDir(pkgPath); err != nil {
		return err
	}
	for _, entry := range entries {
		var fileContent []byte
		if fileContent, err = pkgFiles.ReadFile(fmt.Sprintf("%s/%s", pkgPath, entry.Name())); err != nil {
			return err
		}
		if err = os.MkdirAll(path.Join(dst, pkg), 0700); err != nil {
			return err
		}
		filename := path.Join(dst, pkg, entry.Name())
		if err = os.WriteFile(filename, fileContent, 0600); err != nil {
			return err
		}
	}
	return
}

func tsCopyTo(pkg, dst string) (err error) {

	pkgPath := path.Join("ts", pkg)
	var entries []fs.DirEntry
	if entries, err = tsFiles.ReadDir(pkgPath); err != nil {
		return err
	}
	for _, entry := range entries {
		var fileContent []byte
		if fileContent, err = tsFiles.ReadFile(fmt.Sprintf("%s/%s", pkgPath, entry.Name())); err != nil {
			return err
		}
		if err = os.MkdirAll(path.Join(dst, pkg), 0700); err != nil {
			return err
		}
		filename := path.Join(dst, pkg, entry.Name())
		if err = os.WriteFile(filename, fileContent, 0600); err != nil {
			return err
		}
	}
	return
}

func (tr *Transport) pkgPath(dir string) (pkgPath string) {

	var pkgDir string
	dirAbs, _ := filepath.Abs(dir)
	if modPath, err := mod.GoModPath(dir); err == nil {
		pkgDir = filepath.ToSlash(strings.TrimPrefix(dirAbs, filepath.Dir(modPath)))
	}
	return tr.module.Module.Mod.String() + pkgDir
}
