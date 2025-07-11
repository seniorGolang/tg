// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (srcfile.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/v2/pkg/goimports"
	"github.com/seniorGolang/tg/v2/pkg/utils"
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

	var runner goimports.Runner
	if runner, err = goimports.NewFromFile(filepath); err != nil {
		return
	}

	if err = runner.Run(utils.GetModulePath(filepath)); err != nil {
		return
	}
	return
}
