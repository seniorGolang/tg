// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (service-http.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
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
		Op("*").Id("httpServer"),
		Id("svc").Op("*").Id("server"+svc.Name),
	)

	srcFile.Line().Func().Id("New"+svc.Name).Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id("svc"+svc.Name).Qual(svc.pkgPath, svc.Name)).Params(Id("srv").Op("*").Id("http"+svc.Name)).Block(

		Line().Id("srv").Op("=").Op("&").Id("http"+svc.Name).Values(Dict{
			Id("httpServer"): Op("&").Id("httpServer").Values(Dict{
				Id("log"):                Id("log"),
				Id("maxRequestBodySize"): Id("maxRequestBodySize"),
			}),
			Id("svc"): Id("newServer" + svc.Name).Call(Id("svc" + svc.Name)),
		}),
		Return(),
	)

	srcFile.Line().Func().Params(Id("http").Id("http" + svc.Name)).Id(svc.Name).Params().Params(Id("MiddlewareSet" + svc.Name)).Block(
		Return(Id("http").Dot("svc")),
	)

	srcFile.Line().Add(svc.withLogFunc())
	srcFile.Line().Add(svc.withTraceFunc())

	srcFile.Line().Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id("ServeHTTP").Params(Id("address").String(), Id("options").Op("...").Id("Option")).BlockFunc(func(bg *Group) {

		bg.Line().Id("http").Dot("applyOptions").Call(Id("options").Op("..."))

		bg.Line().Id("route").Op(":=").Qual(packageFastHttpRouter, "New").Call()

		prefix := svc.tags.Value(tagHttpPrefix)

		if svc.tags.Contains(tagServerJsonRPC) {

			bg.Line().Id("route").Dot("POST").Call(Lit(path.Join("/", prefix, svc.lcName())), Id("http").Dot("serveBatch"))

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
				bg.Id("route").Dot(method.httpMethod()).Call(Lit(method.httpPath()), Id("http").Dot("serve"+method.Name))
			}
		}

		bg.Line().Id("http").Dot("log").Dot("WithField").Call(Lit("address"), Id("address")).Dot("Info").Call(Lit(fmt.Sprintf("enable '%s' HTTP transport", svc.Name))).Line()

		bg.Line().Id("http").Dot("srvHttp").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
			Id("ReadTimeout"):        Qual(packageTime, "Second").Op("*").Lit(10),
			Id("Handler"):            Qual(packageCors, "AllowAll").Call().Dot("Handler").Call(Id("route").Dot("Handler")),
			Id("MaxRequestBodySize"): Id("http").Dot("maxRequestBodySize"),
		})

		bg.Line().Go().Func().Params().Block(
			Err().Op(":=").Id("http").Dot("srvHttp").Dot("ListenAndServe").Call(Id("address")),
			Id("ExitOnError").Call(Id("http").Dot("log"), Err(), Lit(fmt.Sprintf("serve '%s' http on ", svc.Name)).Op("+").Id("address")),
		).Call()
	})

	return srcFile.Save(path.Join(outDir, svc.lcName()+"-http.go"))
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
