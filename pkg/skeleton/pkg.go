package skeleton

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
)

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
