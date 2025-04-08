// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-test.go at 24.06.2020, 14:10) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (svc *service) renderTest(outDir string) (err error) {

	outDir, _ = filepath.Abs(outDir)
	srcFile := newSrc(filepath.Base(outDir))
	srcFile.ImportName(packageTesting, "testing")

	for _, method := range svc.methods {

		srcFile.Line().Func().Id(fmt.Sprintf("Test%s%s", svc.Name, method.Name)).Params(Id("t").Op("*").Qual(packageTesting, "T")).Block(
			Id("t").Dot("Error").Call(Lit(fmt.Sprintf("test %s.%s is not implemented", svc.Name, method.Name))),
		)
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"_test.go"))
}
