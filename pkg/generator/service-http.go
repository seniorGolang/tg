// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (service-http.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (svc *service) renderHTTP(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageCors, "cors")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageFastHttp, "fasthttp")
	srcFile.ImportName(packageFastHttpRouter, "router")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("http"+svc.Name).Struct(
		Id("log").Qual(packageLogrus, "FieldLogger"),
		Id("errorHandler").Id("ErrorHandler"),
		Id("svc").Op("*").Id("server"+svc.Name),
		Id("base").Qual(svc.pkgPath, svc.Name),
	)

	srcFile.Line().Func().Id("New"+svc.Name).Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id("svc"+svc.Name).Qual(svc.pkgPath, svc.Name)).Params(Id("srv").Op("*").Id("http"+svc.Name)).Block(

		Line().Id("srv").Op("=").Op("&").Id("http"+svc.Name).Values(Dict{
			Id("log"):  Id("log"),
			Id("base"): Id("svc" + svc.Name),
			Id("svc"):  Id("newServer" + svc.Name).Call(Id("svc" + svc.Name)),
		}),
		Return(),
	)

	srcFile.Line().Func().Params(Id("http").Id("http" + svc.Name)).Id("Service").Params().Params(Id("MiddlewareSet" + svc.Name)).Block(
		Return(Id("http").Dot("svc")),
	)

	srcFile.Line().Add(svc.withLogFunc())
	srcFile.Line().Add(svc.withTraceFunc())
	srcFile.Line().Add(svc.withMetricsFunc())
	srcFile.Line().Add(svc.withErrorHandler())

	srcFile.Line().Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("SetRoutes").Params(Id("route").Op("*").Qual(packageFastHttpRouter, "Router")).BlockFunc(func(bg *Group) {

		if svc.tags.Contains(tagServerJsonRPC) {

			bg.Line().Id("route").Dot("POST").Call(Lit(path.Join("/", svc.lccName())), Id("http").Dot("serveBatch"))

			for _, method := range svc.methods {

				if !method.isJsonRPC() {
					continue
				}
				bg.Id("route").Dot("POST").Call(Lit(method.jsonrpcPath()), Id("http").Dot("serve"+method.Name))
			}

		}

		if svc.tags.Contains(tagServerHTTP) {

			bg.Line()

			for _, method := range svc.methods {

				if !method.isHTTP() {
					continue
				}
				if method.tags.Contains(tagHandler) {
					bg.Id("route").Dot(method.httpMethod()).Call(Lit(method.httpPath()), Func().Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
						Qual(method.handlerQual()).Call(Id(_ctx_), Id("http").Dot("svc")),
					))
					continue
				}
				bg.Id("route").Dot(method.httpMethod()).Call(Lit(method.httpPath()), Id("http").Dot("serve"+method.Name))
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

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("WithLog").Params(Id("log").Qual(packageLogrus, "FieldLogger")).Params(Op("*").Id("http" + svc.Name)).BlockFunc(func(bg *Group) {

		bg.Id("http").Dot("svc").Dot("WithLog").Call(Id("log"))
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
