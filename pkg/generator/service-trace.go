// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-trace.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

func (svc *service) renderTrace(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("trace" + svc.Name).Struct(
		Id("next").Qual(svc.pkgPath, svc.Name),
	)

	srcFile.Line().Func().Id("traceMiddleware" + svc.Name).Params(Id("next").Qual(svc.pkgPath, svc.Name)).Params(Qual(svc.pkgPath, svc.Name)).Block(
		Return(Op("&").Id("trace" + svc.Name).Values(Dict{
			Id("next"): Id("next"),
		})),
	)

	for _, method := range svc.methods {
		srcFile.Line().Func().Params(Id("svc").Id("trace"+svc.Name)).Id(method.Name).Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).Block(

			Line(),
			Var().Id("span").Qual(packageTrace, "Span"),
			List(Id(_ctx_), Id("span")).Op("=").
				Qual(packageOTEL, "Tracer").
				Call(Qual(packageFmt, "Sprintf").Call(Lit("tg:%s"), Id("VersionTg"))).Dot("Start").Call(Id(_ctx_), Lit(method.fullName())),
			Defer().Func().Params().Block(
				Id("span").Dot("RecordError").Call(Err()),
				Id("span").Dot("End").Call(),
			).Call(),
			Return(Id("svc").Dot("next").Dot(method.Name).CallFunc(func(cg *Group) {
				for _, arg := range method.Args {
					argCode := Id(arg.Name)
					if types.IsEllipsis(arg.Type) {
						argCode.Op("...")
					}
					cg.Add(argCode)
				}
			})),
		)
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-trace.go"))
}
