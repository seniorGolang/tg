// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (srcfile.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"os/exec"

	"github.com/dave/jennifer/jen"
)

type goFile struct {
	*jen.File
	filepath string
}

func newSrc(pkgName string) goFile {
	return goFile{
		File: jen.NewFile(pkgName),
	}
}

func (src *goFile) Save(filepath string) (err error) {

	src.filepath = filepath
	if err = src.File.Save(src.filepath); err != nil {
		return
	}

	if err = src.goImports(); err != nil {
		return
	}
	return
}

func (src *goFile) goImports() (err error) {

	var execPath string
	if execPath, err = exec.LookPath("goimports"); err != nil {
		return nil
	}
	return exec.Command(execPath, "-local", "-w", src.filepath).Run()
}
