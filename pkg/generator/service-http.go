// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-http.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderHTTP(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageCors, "cors")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("http"+svc.Name).Struct(
		Id("errorHandler").Id("ErrorHandler"),
		Id("maxBatchSize").Int(),
		Id("maxParallelBatch").Int(),
		Id("svc").Op("*").Id("server"+svc.Name),
		Id("base").Qual(svc.pkgPath, svc.Name),
	)

	srcFile.Line().Func().Id("New"+svc.Name).Params(Id("svc"+svc.Name).Qual(svc.pkgPath, svc.Name)).Params(Id("srv").Op("*").Id("http"+svc.Name)).Block(
		Line().Id("srv").Op("=").Op("&").Id("http"+svc.Name).Values(Dict{
			Id("base"): Id("svc" + svc.Name),
			Id("svc"):  Id("newServer" + svc.Name).Call(Id("svc" + svc.Name)),
		}),
		Return(),
	)
	srcFile.Line().Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("Service").Params().Params(Op("*").Id("server" + svc.Name)).Block(
		Return(Id("http").Dot("svc")),
	)
	srcFile.Line().Add(svc.withLogFunc())
	if svc.tags.IsSet(tagTrace) {
		srcFile.Line().Add(svc.withTraceFunc())
	}
	if svc.tags.IsSet(tagMetrics) {
		srcFile.Line().Add(svc.withMetricsFunc())
	}
	srcFile.Line().Add(svc.withErrorHandler())

	srcFile.Line().Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("SetRoutes").Params(Id("route").Op("*").Qual(packageFiber, "App")).BlockFunc(func(bg *Group) {
		if svc.tags.Contains(tagServerJsonRPC) {
			bg.Id("route").Dot("Post").Call(Lit(svc.batchPath()), Id("http").Dot("serveBatch"))
			for _, method := range svc.methods {
				if !method.isJsonRPC() {
					continue
				}
				bg.Id("route").Dot("Post").Call(Lit(method.jsonrpcPath()), Id("http").Dot("serve"+method.Name))
			}
		}
		if svc.tags.Contains(tagServerHTTP) {
			for _, method := range svc.methods {
				if !method.isHTTP() {
					continue
				}
				if method.tags.Contains(tagHandler) {
					bg.Id("route").Dot(utils.ToCamel(method.httpMethod())).Call(Lit(method.httpPath()), Func().Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Err().Error()).Block(
						Return().Qual(method.handlerQual()).Call(Id(_ctx_), Id("http").Dot("base")),
					))
					continue
				}
				bg.Id("route").Dot(utils.ToCamel(method.httpMethod())).Call(Lit(method.httpPath()), Id("http").Dot("serve"+method.Name))
			}
		}
	})
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-http.go"))
}

func (svc *service) withErrorHandler() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("WithErrorHandler").Params(Id("handler").Id("ErrorHandler")).Params(Op("*").Id("http" + svc.Name)).BlockFunc(func(bg *Group) {

		bg.Id("http").Dot("errorHandler").Op("=").Id("handler")
		bg.Return(Id("http"))
	})
}

func (svc *service) withLogFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("WithLog").Params().Params(Op("*").Id("http" + svc.Name)).BlockFunc(func(bg *Group) {

		bg.Id("http").Dot("svc").Dot("WithLog").Call()
		bg.Return(Id("http"))
	})
}

func (svc *service) withTraceFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("WithTrace").Params().Params(Op("*").Id("http" + svc.Name)).BlockFunc(func(bg *Group) {

		bg.Id("http").Dot("svc").Dot("WithTrace").Call()
		bg.Return(Id("http"))
	})
}

func (svc *service) withMetricsFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("WithMetrics").Params().Params(Op("*").Id("http" + svc.Name)).BlockFunc(func(bg *Group) {

		bg.Id("http").Dot("svc").Dot("WithMetrics").Call()
		bg.Return(Id("http"))
	})
}
