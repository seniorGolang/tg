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
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")
	//srcFile.ImportName(packageFastHttpAdapt, "fasthttpadaptor")

	srcFile.Line().Const().Id("maxRequestBodySize").Op("=").Lit(100 * 1024 * 1024)

	for _, service := range tr.services {
		srcFile.ImportName(service.pkgPath, filepath.Base(service.pkgPath))
	}

	srcFile.Line().Add(tr.serverType())
	srcFile.Line().Add(tr.serverNewFunc())

	srcFile.Line().Add(tr.fiberFunc())
	srcFile.Line().Add(tr.withLogFunc())
	srcFile.Line().Add(tr.withTraceFunc())
	srcFile.Line().Add(tr.withMetricsFunc())

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

func (tr Transport) fiberFunc() Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Fiber").Params().Params(Op("*").Qual(packageFiber, "App")).Block(
		Return(Id("srv").Dot("srvHTTP")),
	)
}

func (tr Transport) withLogFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithLog").Params(Id("log").Qual(packageLogrus, "FieldLogger")).Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for serviceName := range tr.services {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithLog").Call(Id("log")),
			)
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) withTraceFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithTrace").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for serviceName := range tr.services {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithTrace").Call(),
			)
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) withMetricsFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithMetrics").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for serviceName := range tr.services {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithMetrics").Call(),
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

		g.Line().Id("srvHTTP").Op("*").Qual(packageFiber, "App")
		g.Id("srvHealth").Op("*").Qual(packageFiber, "App")
		g.Id("srvPPROF").Op("*").Qual(packageFiber, "App")

		g.Line().Id("reporterCloser").Qual(packageIO, "Closer")

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
				Id("maxRequestBodySize"): Id("maxRequestBodySize"),
				Id("srvHTTP"):             Qual(packageFiber, "New").Call(),
			})
			if tr.hasJsonRPC {
				bg.Id("srv").Dot("srvHTTP").Dot("Post").Call(Lit("/"), Id("srv").Dot("serveBatch"))
			}
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			bg.Return()
		})
}

func (tr Transport) serveHealthFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHealth").Params(Id("address").String(), Id("response").Interface()).Block(

		Id("srv").Dot("srvHealth").Op("=").Qual(packageFiber, "New").Call(),
		Id("srv").Dot("srvHealth").Dot("Get").Call(Lit("/"),
			Func().Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Error()).Block(
				Return().Id(_ctx_).Dot("JSON").Call(Id("response")),
			)),
		Go().Func().Params().Block(
			Err().Op(":=").Id("srv").Dot("srvHealth").Dot("Listen").Call(Id("address")),
			Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve health on ").Op("+").Id("address")),
		).Call(),
	)
}

func (tr Transport) shutdownFunc() Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Shutdown").Params().Block(
		If(Id("srv").Dot("srvHTTP").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHTTP").Dot("Shutdown").Call(),
		),
		If(Id("srv").Dot("srvHealth").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHealth").Dot("Shutdown").Call(),
		),
		If(Id("srvMetrics").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srvMetrics").Dot("Shutdown").Call(),
		),
		If(Id("srv").Dot("srvPPROF").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvPPROF").Dot("Shutdown").Call(),
		),
	)
}

func (tr Transport) serveProfileFunc() Code {

	//return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServePPROF").Params(Id("address").String()).Block(
	//
	//	Line().Qual(packageRuntime, "SetBlockProfileRate").Call(Lit(1)),
	//	Qual(packageRuntime, "SetMutexProfileFraction").Call(Lit(5)),
	//
	//	Line().Id("srv").Dot("srvPPROF").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
	//		Id("ReadTimeout"): Qual(packageTime, "Second").Op("*").Lit(10),
	//		Id("Handler"):     Qual(packageFastHttpAdapt, "NewFastHTTPHandler").Call(Qual(packageHttp, "DefaultServeMux")),
	//	}),
	//
	//	Line().Go().Func().Params().Block(
	//		Err().Op(":=").Id("srv").Dot("srvPPROF").Dot("ListenAndServe").Call(Id("address")),
	//		Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve PPROF on ").Op("+").Id("address")),
	//	).Call(),
	//)
	return nil
}

func (tr Transport) sendResponseFunc() Code {
	return Func().Id("sendResponse").Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("resp").Interface()).Block(
		Id(_ctx_).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit("application/json")),
		If(Err().Op(":=").Qual(packageJson, "NewEncoder").Call(Id(_ctx_)).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
			Id("log").Dot("WithField").Call(Lit("body"), Id(_ctx_).Dot("Body").Call()).Dot("WithError").Call(Err()).Dot("Error").Call(Lit("response write error")),
		),
	)
}
