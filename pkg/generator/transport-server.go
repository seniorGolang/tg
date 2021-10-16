// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-server.go at 15.06.2020, 11:38) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderServer(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageIO, "io")
	srcFile.ImportName(packageJson, "json")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")

	srcFile.Line().Const().Id("maxRequestBodySize").Op("=").Lit(100 * 1024 * 1024)

	var hasTrace, hasMetrics bool
	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		if svc.tags.IsSet(tagTrace) {
			hasTrace = true
		}
		if svc.tags.IsSet(tagMetrics) {
			hasMetrics = true
		}
		srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	}

	srcFile.Const().Id("headerRequestID").Op("=").Lit("X-Request-Id")

	srcFile.Line().Add(tr.serverType())
	srcFile.Line().Add(tr.serverNewFunc())
	srcFile.Line().Add(tr.fiberFunc())
	srcFile.Line().Add(tr.withLogFunc())
	srcFile.Line().Add(tr.serveHealthFunc())
	srcFile.Line().Add(tr.sendResponseFunc())
	srcFile.Line().Add(tr.shutdownFunc(hasMetrics))
	if hasTrace {
		srcFile.Line().Add(tr.withTraceFunc())
	}
	if hasMetrics {
		srcFile.Line().Add(tr.withMetricsFunc())
	}

	for _, serviceName := range tr.serviceKeys() {
		srcFile.Line().Add(Func().Params(Id("srv").Id("Server")).Id(serviceName).Params().Params(Op("*").Id("http" + serviceName)).Block(
			Return(Id("srv").Dot("http" + serviceName)),
		))
	}

	return srcFile.Save(path.Join(outDir, "server.go"))
}

func (tr Transport) fiberFunc() Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Fiber").Params().Params(Op("*").Qual(packageFiber, "App")).Block(
		Return(Id("srv").Dot("srvHTTP")),
	)
}

func (tr Transport) withLogFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithLog").Params(Id("log").Qual(packageZeroLog, "Logger")).Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for _, serviceName := range tr.serviceKeys() {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithLog").Call(Id("log")),
			)
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) withTraceFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithTrace").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for _, serviceName := range tr.serviceKeys() {
			svc := tr.services[serviceName]
			if svc.tags.IsSet(tagTrace) {
				bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
					Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithTrace").Call(),
				)
			}
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) withMetricsFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithMetrics").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for _, serviceName := range tr.serviceKeys() {
			svc := tr.services[serviceName]
			if svc.tags.IsSet(tagMetrics) {
				bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
					Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithMetrics").Call(),
				)
			}
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) serverType() Code {

	return Type().Id("Server").StructFunc(func(g *Group) {
		g.Id("log").Qual(packageZeroLog, "Logger")
		g.Line().Id("httpAfter").Op("[]").Id("Handler")
		g.Id("httpBefore").Op("[]").Id("Handler")
		g.Line().Id("config").Qual(packageFiber, "Config")
		g.Line().Id("srvHTTP").Op("*").Qual(packageFiber, "App")
		g.Id("srvHealth").Op("*").Qual(packageFiber, "App")
		g.Line().Id("reporterCloser").Qual(packageIO, "Closer")
		for _, serviceName := range tr.serviceKeys() {
			g.Id("http" + serviceName).Op("*").Id("http" + serviceName)
		}
	})
}

func (tr Transport) serverNewFunc() Code {

	return Func().Id("New").Params(Id("log").Qual(packageZeroLog, "Logger"), Id("options").Op("...").Id("Option")).Params(Id("srv").Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.Line().Id("srv").Op("=").Op("&").Id("Server").Values(Dict{
				Id("log"): Id("log"),
				Id("config"): Qual(packageFiber, "Config").Values(Dict{
					Id("DisableStartupMessage"): True(),
					Id("BodyLimit"):             Id("maxRequestBodySize"),
				}),
			})
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			bg.Id("srv").Dot("srvHTTP").Op("=").Qual(packageFiber, "New").Call(Id("srv").Dot("config"))
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			if tr.hasJsonRPC {
				bg.Id("srv").Dot("srvHTTP").Dot("Post").Call(Lit("/"+tr.tags.Value(tagHttpPrefix, "")), Id("srv").Dot("serveBatch"))
			}
			bg.Return()
		})
}

func (tr Transport) serveHealthFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHealth").Params(Id("address").String(), Id("response").Interface()).Block(

		Id("srv").Dot("srvHealth").Op("=").Qual(packageFiber, "New").Call(Qual(packageFiber, "Config").Values(Dict{Id("DisableStartupMessage"): True()})),
		Id("srv").Dot("srvHealth").Dot("Get").Call(Lit("/health"),
			Func().Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Error()).Block(
				Return().Id(_ctx_).Dot("JSON").Call(Id("response")),
			)),
		Go().Func().Params().Block(
			Err().Op(":=").Id("srv").Dot("srvHealth").Dot("Listen").Call(Id("address")),
			Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve health on ").Op("+").Id("address")),
		).Call(),
	)
}

func (tr Transport) shutdownFunc(hasMetrics bool) Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Shutdown").Params().BlockFunc(func(bg *Group) {
		bg.If(Id("srv").Dot("srvHTTP").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHTTP").Dot("Shutdown").Call(),
		)
		bg.If(Id("srv").Dot("srvHealth").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHealth").Dot("Shutdown").Call(),
		)
		if hasMetrics {
			bg.If(Id("srvMetrics").Op("!=").Id("nil")).Block(
				Id("_").Op("=").Id("srvMetrics").Dot("Shutdown").Call(),
			)
		}
	})
}

func (tr Transport) sendResponseFunc() Code {
	return Func().Id("sendResponse").Params(Id("log").Qual(packageZeroLog, "Logger"), Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("resp").Interface()).Params(Err().Error()).Block(
		Id(_ctx_).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit("application/json")),
		If(Err().Op("=").Qual(packageJson, "NewEncoder").Call(Id(_ctx_)).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
			Id("log").Dot("Error").Call().Dot("Err").Call(Err()).Dot("Str").Call(Lit("body"), String().Call(Id(_ctx_).Dot("Body").Call())).Dot("Msg").Call(Lit("response write error")),
		),
		Return(),
	)
}
