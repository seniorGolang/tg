// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (mod.go at 14.05.2020, 2:20) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package mod

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/rogpeppe/go-internal/modfile"
	"golang.org/x/mod/module"
)

func PkgModPath(pkgName string) string {

	modPath, _ := goModPath(".")
	modInfo := parseMod(modPath)
	pkgTokens := strings.Split(pkgName, "/")
	for i := 0; i < len(pkgTokens); i++ {
		pathTry := strings.Join(pkgTokens[:len(pkgTokens)-i], "/")
		for modPkg, modPath := range modInfo {
			if pathTry == modPkg {
				esc, _ := module.EscapePath(modPkg)
				modPath = strings.Replace(modPath, modPkg, esc, 1)
				if len(strings.Split(modPkg, "/")) == 1 {
					return path.Join(modPath, strings.Join(pkgTokens, "/"))
				}
				return path.Join(modPath, strings.Join(pkgTokens[len(pkgTokens)-i:], "/"))
			}
		}
	}
	return ""
}

func parseMod(modPath string) (pkgPath map[string]string) {

	var err error
	var fileBytes []byte

	if fileBytes, err = ioutil.ReadFile(modPath); err != nil {
		return
	}

	mod, err := modfile.Parse(modPath, fileBytes, nil)
	if err != nil {
		return
	}

	goPath := os.Getenv("GOPATH")

	pkgPath = make(map[string]string)
	pkgPath[mod.Module.Mod.Path] = path.Dir(modPath)

	for _, require := range mod.Require {
		pkgPath[require.Syntax.Token[0]] = fmt.Sprintf("%s/pkg/mod/%s@%s", goPath, require.Syntax.Token[0], require.Mod.Version)
	}
	return
}

// empty if no go.mod, GO111MODULE=off or go without go modules support
func goModPath(root string) (string, error) {

	var stdout []byte
	var err error
	for {
		cmd := exec.Command("go", "env", "GOMOD")
		cmd.Dir = root
		stdout, err = cmd.Output()
		if err == nil {
			break
		}
		if _, ok := err.(*os.PathError); ok {
			// try to find go.mod on level higher
			r := filepath.Join(root, "..")
			if r == root { // when we in root directory stop trying
				return "", err
			}
			root = r
			continue
		}
		return "", err
	}
	goModPath := string(bytes.TrimSpace(stdout))
	return goModPath, nil
}
