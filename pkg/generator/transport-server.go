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

	srcFile.Anon(packagePPROF)

	srcFile.ImportName(packageIO, "io")
	srcFile.ImportName(packageGotils, "gotils")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageFastHttp, "fasthttp")
	srcFile.ImportName(packageFastHttpRouter, "router")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")
	srcFile.ImportName(packageFastHttpAdapt, "fasthttpadaptor")

	srcFile.Line().Const().Id("maxRequestBodySize").Op("=").Lit(100 * 1024 * 1024)
	srcFile.Line().Type().Id("middleware").Func().Params(Qual(packageFastHttp, "RequestHandler")).Params(Qual(packageFastHttp, "RequestHandler"))

	for _, service := range tr.services {
		srcFile.ImportName(service.pkgPath, filepath.Base(service.pkgPath))
	}

	srcFile.Line().Add(tr.serverType())
	srcFile.Line().Add(tr.serverNewFunc())

	srcFile.Line().Add(tr.serveHTTP())
	srcFile.Line().Add(tr.serveHTTPS())
	srcFile.Line().Add(tr.httpHandler())

	srcFile.Line().Add(tr.routerFunc())
	srcFile.Line().Add(tr.withLogFunc())
	srcFile.Line().Add(tr.withTraceFunc())

	srcFile.Line().Add(tr.serveProfileFunc())
	srcFile.Line().Add(tr.serveHealthFunc())

	srcFile.Line().Add(tr.shutdownFunc())

	srcFile.Line().Add(tr.sendResponseFunc())

	for serviceName := range tr.services {
		srcFile.Line().Add(Func().Params(Id("srv").Id("Server")).Id(serviceName).Params().Params(Op("*").Id("http" + serviceName)).Block(
			Return(Id("srv").Dot("http" + serviceName)),
		))
	}

	return srcFile.Save(path.Join(outDir, "server.go"))
}

func (tr Transport) routerFunc() Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Router").Params().Params(Op("*").Qual(packageFastHttpRouter, "Router")).Block(
		Return(Id("srv").Dot("router")),
	)
}

func (tr Transport) withLogFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithLog").Params(Id("log").Qual(packageLogrus, "FieldLogger")).Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for serviceName := range tr.services {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot(serviceName).Call().Dot("WithLog").Call(Id("log")),
			)
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) withTraceFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithTrace").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for serviceName := range tr.services {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot(serviceName).Call().Dot("WithTrace").Call(),
			)
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) serverType() Code {

	return Type().Id("Server").StructFunc(func(g *Group) {

		g.Id("log").Qual(packageLogrus, "FieldLogger")

		g.Line().Id("httpAfter").Op("[]").Id("Handler")
		g.Id("httpBefore").Op("[]").Id("Handler")
		g.Line().Id("maxRequestBodySize").Int()

		g.Line().Id("srvHTTP").Op("*").Qual(packageFastHttp, "Server")
		g.Id("srvHealth").Op("*").Qual(packageFastHttp, "Server")
		g.Id("srvPPROF").Op("*").Qual(packageFastHttp, "Server")

		g.Line().Id("reporterCloser").Qual(packageIO, "Closer")

		g.Line().Id("router").Op("*").Qual(packageFastHttpRouter, "Router").Line()

		for serviceName := range tr.services {
			g.Id("http" + serviceName).Op("*").Id("http" + serviceName)
		}
	})
}

func (tr Transport) serverNewFunc() Code {

	return Func().Id("New").Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id("options").Op("...").Id("Option")).Params(Id("srv").Op("*").Id("Server")).

		BlockFunc(func(bg *Group) {
			bg.Line().Id("srv").Op("=").Op("&").Id("Server").Values(Dict{
				Id("log"):                Id("log"),
				Id("router"):             Qual(packageFastHttpRouter, "New").Call(),
				Id("maxRequestBodySize"): Id("maxRequestBodySize"),
			})
			if tr.hasJsonRPC {
				bg.Id("srv").Dot("router").Dot("POST").Call(Lit("/"), Id("srv").Dot("serveBatch"))
			}
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			bg.Return()
		})
}

func (tr Transport) serveHTTP() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHTTP").Params(Id("address").String(), Id("wraps").Op("...").Id("middleware")).BlockFunc(

		func(bg *Group) {

			bg.Line().Id("srv").Dot("log").Dot("WithField").Call(Lit("address"), Id("address")).Dot("Info").Call(Lit("enable HTTP transport"))

			bg.Id("handler").Op(":=").Id("srv").Dot("httpHandler").Call()

			bg.Line().For(List(Id("_"), Id("wrap")).Op(":=").Range().Id("wraps")).Block(
				Id("handler").Op("=").Id("wrap").Call(Id("handler")),
			)
			bg.Id("srv").Dot("srvHTTP").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
				Id("ReadTimeout"):        Qual(packageTime, "Second").Op("*").Lit(10),
				Id("Handler"):            Id("handler"),
				Id("MaxRequestBodySize"): Id("srv").Dot("maxRequestBodySize"),
			})
			bg.Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvHTTP").Dot("ListenAndServe").Call(Id("address")),
				Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve http on ").Op("+").Id("address")),
			).Call()
		},
	)
}

func (tr Transport) serveHTTPS() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHTTPS").Params(Id("address"), Id("certFile"), Id("keyFile").String(), Id("wraps").Op("...").Id("middleware")).BlockFunc(

		func(bg *Group) {

			bg.Line().Id("srv").Dot("log").Dot("WithField").Call(Lit("address"), Id("address")).Dot("Info").Call(Lit("enable HTTP transport"))

			bg.Id("handler").Op(":=").Id("srv").Dot("httpHandler").Call()

			bg.Line().For(List(Id("_"), Id("wrap")).Op(":=").Range().Id("wraps")).Block(
				Id("handler").Op("=").Id("wrap").Call(Id("handler")),
			)
			bg.Id("srv").Dot("srvHTTP").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
				Id("ReadTimeout"):        Qual(packageTime, "Second").Op("*").Lit(10),
				Id("Handler"):            Id("handler"),
				Id("MaxRequestBodySize"): Id("srv").Dot("maxRequestBodySize"),
			})
			bg.Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvHTTP").Dot("ListenAndServeTLS").Call(Id("address"), Id("certFile"), Id("keyFile")),
				Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve http on ").Op("+").Id("address")),
			).Call()
		},
	)
}

func (tr Transport) httpHandler() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("httpHandler").Params().Params(Qual(packageFastHttp, "RequestHandler")).Block(

		Return().Func().Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
			Line().For(List(Id("_"), Id("before")).Op(":=").Range().Id("srv").Dot("httpBefore")).Block(
				Id("before").Call(Id("ctx")),
			),
			Id("srv").Dot("router").Dot("Handler").Call(Id(_ctx_)),
			Line().For(List(Id("_"), Id("after")).Op(":=").Range().Id("srv").Dot("httpAfter")).Block(
				Id("after").Call(Id("ctx")),
			),
		),
	)
}

func (tr Transport) serveHealthFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHealth").Params(Id("address").String()).Block(

		Line().Id("srv").Dot("srvHealth").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
			Id("ReadTimeout"): Qual(packageTime, "Second").Op("*").Lit(10),
			Id("Handler"): Func().Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
				Id(_ctx_).Dot("SetStatusCode").Call(Qual(packageFastHttp, "StatusOK")),
				List(Id("_"), Id("_")).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("ok")),
			),
		}),
		Go().Func().Params().Block(
			Err().Op(":=").Id("srv").Dot("srvHealth").Dot("ListenAndServe").Call(Id("address")),
			Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve health on ").Op("+").Id("address")),
		).Call(),
	)
}

func (tr Transport) shutdownFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Shutdown").Params().Block(

		Line().If(Id("srv").Dot("srvHTTP").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHTTP").Dot("Shutdown").Call(),
		),

		Line().If(Id("srv").Dot("srvHealth").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHealth").Dot("Shutdown").Call(),
		),

		Line().If(Id("srvMetrics").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srvMetrics").Dot("Shutdown").Call(),
		),

		Line().If(Id("srv").Dot("srvPPROF").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvPPROF").Dot("Shutdown").Call(),
		),
	)
}

func (tr Transport) serveProfileFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServePPROF").Params(Id("address").String()).Block(

		Line().Qual(packageRuntime, "SetBlockProfileRate").Call(Lit(1)),
		Qual(packageRuntime, "SetMutexProfileFraction").Call(Lit(5)),

		Line().Id("srv").Dot("srvPPROF").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
			Id("ReadTimeout"): Qual(packageTime, "Second").Op("*").Lit(10),
			Id("Handler"):     Qual(packageFastHttpAdapt, "NewFastHTTPHandler").Call(Qual(packageHttp, "DefaultServeMux")),
		}),

		Line().Go().Func().Params().Block(
			Err().Op(":=").Id("srv").Dot("srvPPROF").Dot("ListenAndServe").Call(Id("address")),
			Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve PPROF on ").Op("+").Id("address")),
		).Call(),
	)
}

func (tr Transport) sendResponseFunc() Code {

	return Func().Id("sendResponse").Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx"), Id("resp").Interface()).Block(

		Line().Id(_ctx_).Dot("SetContentType").Call(Lit("application/json")),

		Line().If(Err().Op(":=").Qual(packageJson, "NewEncoder").Call(Id(_ctx_)).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
			Id("log").Dot("WithField").Call(Lit("body"), Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("PostBody").Call())).Dot("WithError").Call(Err()).Dot("Error").Call(Lit("response write error")),
		),
	)
}
