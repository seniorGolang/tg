// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (srcfile.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"os/exec"

	"github.com/dave/jennifer/jen"
)

type srcFile struct {
	*jen.File
	filepath string
}

func newSrc(pkgName string) srcFile {
	return srcFile{
		File: jen.NewFile(pkgName),
	}
}

func (src *srcFile) Save(filepath string) (err error) {

	src.filepath = filepath
	if err = src.File.Save(src.filepath); err != nil {
		return
	}

	if err = src.goImports(); err != nil {
		return
	}
	return
}

func (src srcFile) goImports() (err error) {

	var execPath string
	if execPath, err = exec.LookPath("goimports"); err != nil {
		return nil
	}
	return exec.Command(execPath, "-local", "-w", src.filepath).Run()
}
