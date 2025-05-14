// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-middleware.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (svc *service) renderMiddleware(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	for _, method := range svc.methods {
		srcFile.Type().Id(svc.Name + method.Name).Func().Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results))
	}

	srcFile.Line().Type().Id("Middleware" + svc.Name).Func().Params(Id("next").Qual(svc.pkgPath, svc.Name)).Params(Qual(svc.pkgPath, svc.Name)).Line()

	for _, method := range svc.methods {
		srcFile.Type().Id("Middleware" + svc.Name + method.Name).Func().Params(Id("next").Id(svc.Name + method.Name)).Params(Id(svc.Name + method.Name))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-middleware.go"))
}
