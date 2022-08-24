// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (pkgpath.go at 14.05.2020, 2:21) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"bytes"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/seniorGolang/tg/v2/pkg/logger"
)

var (
	log = logger.Log.WithField("module", "server")
)

func GetPkgPath(fName string, isDir bool) (string, error) {

	// find go.mod file
	goModPath, err := GoModPath(fName, isDir)
	if err != nil {
		log.Error(errors.Wrap(err, "cannot find go.mod because of"))
	}

	if strings.Contains(goModPath, "go.mod") {
		pkgPath, err := GetPkgPathFromGoMod(fName, isDir, goModPath)
		if err != nil {
			return "", err
		}
		return pkgPath, nil
	}
	return GetPkgPathFromGOPATH(fName, isDir)
}

var (
	goModPathCache = make(map[string]string)
)

func GoModPath(fName string, isDir bool) (string, error) {

	root := fName

	if !isDir {
		root = filepath.Dir(fName)
	}

	goModPath, ok := goModPathCache[root]

	if ok {
		return goModPath, nil
	}

	defer func() {
		goModPathCache[root] = goModPath
	}()

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
	goModPath = string(bytes.TrimSpace(stdout))
	return goModPath, nil
}

func GetPkgPathFromGoMod(fName string, isDir bool, goModPath string) (string, error) {

	modulePath := GetModulePath(goModPath)

	if modulePath == "" {
		return "", errors.Errorf("cannot determine module path from %s", goModPath)
	}

	rel := path.Join(modulePath, filePathToPackagePath(strings.TrimPrefix(fName, filepath.Dir(goModPath))))

	if !isDir {
		return path.Dir(rel), nil
	}
	return path.Clean(rel), nil
}

var (
	gopathCache           = ""
	modulePrefix          = []byte("\nmodule ")
	pkgPathFromGoModCache = make(map[string]string)
)

func GetModulePath(goModPath string) string {

	pkgPath, ok := pkgPathFromGoModCache[goModPath]

	if ok {
		return pkgPath
	}

	defer func() {
		pkgPathFromGoModCache[goModPath] = pkgPath
	}()

	data, err := os.ReadFile(goModPath)

	if err != nil {
		return ""
	}

	var i int

	if bytes.HasPrefix(data, modulePrefix[1:]) {
		i = 0
	} else {
		i = bytes.Index(data, modulePrefix)
		if i < 0 {
			return ""
		}
		i++
	}

	line := data[i:]

	// Cut line at \n, drop trailing \r if present.
	if j := bytes.IndexByte(line, '\n'); j >= 0 {
		line = line[:j]
	}

	if line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	line = line[len("module "):]

	// If quoted, unquote.
	pkgPath = strings.TrimSpace(string(line))

	if pkgPath != "" && pkgPath[0] == '"' {
		s, err := strconv.Unquote(pkgPath)
		if err != nil {
			return ""
		}
		pkgPath = s
	}
	return pkgPath
}

func GetPkgPathFromGOPATH(fName string, isDir bool) (string, error) {

	if gopathCache == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			var err error
			gopath, err = GetDefaultGoPath()
			if err != nil {
				return "", errors.Wrap(err, "cannot determine GOPATH")
			}
		}
		gopathCache = gopath
	}

	for _, p := range filepath.SplitList(gopathCache) {
		prefix := filepath.Join(p, "src") + string(filepath.Separator)
		if rel := strings.TrimPrefix(fName, prefix); rel != fName {
			if !isDir {
				return path.Dir(filePathToPackagePath(rel)), nil
			} else {
				return path.Clean(filePathToPackagePath(rel)), nil
			}
		}
	}

	return "", errors.Errorf("file '%s' is not in GOPATH. Checked paths:\n%s", fName, strings.Join(filepath.SplitList(gopathCache), "\n"))
}

func filePathToPackagePath(path string) string {
	return filepath.ToSlash(path)
}

func GetDefaultGoPath() (string, error) {

	if build.Default.GOPATH != "" {
		return build.Default.GOPATH, nil
	}

	output, err := exec.Command("go", "env", "GOPATH").Output()
	return string(bytes.TrimSpace(output)), err
}
