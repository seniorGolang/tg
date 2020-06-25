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
	srcFile.ImportName(packageCors, "cors")
	srcFile.ImportName(packageGotils, "gotils")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageFastHttp, "fasthttp")
	srcFile.ImportName(packageFastHttpRouter, "router")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")
	srcFile.ImportName(packageFastHttpAdapt, "fasthttpadaptor")

	srcFile.Line().Const().Id("maxRequestBodySize").Op("=").Lit(100 * 1024 * 1024)

	for _, service := range tr.services {
		srcFile.ImportName(service.pkgPath, filepath.Base(service.pkgPath))
	}

	srcFile.Line().Add(tr.serverType())
	srcFile.Line().Add(tr.serverNewFunc()).Line()

	for serviceName := range tr.services {
		srcFile.Line().Add(Func().Params(Id("srv").Id("Server")).Id(serviceName).Params().Params(Id("MiddlewareSet" + serviceName)).Block(
			Return(Id("srv").Dot("http" + serviceName).Dot("svc")),
		))
	}

	srcFile.Line().Add(tr.serveHTTP())
	srcFile.Line().Add(tr.serveHTTPS())

	srcFile.Line().Add(tr.routerFunc())
	srcFile.Line().Add(tr.withLogFunc())
	srcFile.Line().Add(tr.withTraceFunc())

	srcFile.Line().Add(tr.serveProfileFunc())
	srcFile.Line().Add(tr.serveHealthFunc())

	srcFile.Line().Add(tr.shutdownFunc())

	srcFile.Line().Add(tr.sendResponseFunc())

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
			bg.Id("srv").Dot(serviceName).Call().Dot("WithLog").Call(Id("log"))
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) withTraceFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithTrace").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for serviceName := range tr.services {
			bg.Id("srv").Dot(serviceName).Call().Dot("WithTrace").Call()
		}
		bg.Return(Id("srv"))
	})
}

func (tr Transport) serverType() Code {

	return Type().Id("Server").StructFunc(func(g *Group) {

		g.Op("*").Id("httpServer")

		g.Line().Id("srvHealth").Op("*").Qual(packageFastHttp, "Server")
		g.Id("srvPPROF").Op("*").Qual(packageFastHttp, "Server")
		g.Id("srvMetrics").Op("*").Qual(packageFastHttp, "Server").Line()

		g.Line().Id("reporterCloser").Qual(packageIO, "Closer").Line()

		// route *router.Router
		g.Line().Id("router").Op("*").Qual(packageFastHttpRouter, "Router")

		for serviceName := range tr.services {
			g.Id("http" + serviceName).Op("*").Id("http" + serviceName).Line()
		}
	})
}

func (tr Transport) serverNewFunc() Code {

	return Func().Id("New").
		ParamsFunc(func(pg *Group) {

			pg.Add(Id("log").Qual(packageLogrus, "FieldLogger"))

			for _, serviceName := range tr.serviceKeys() {
				pg.Add(Id("svc"+serviceName).Qual(tr.services[serviceName].pkgPath, serviceName))
			}
		}).
		Params(Id("srv").Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {

			bg.Line().Id("srv").Op("=").Op("&").Id("Server").ValuesFunc(func(sg *Group) {

				values := Dict{
					Id("router"): Qual(packageFastHttpRouter, "New").Call(),
					Id("httpServer"): Op("&").Id("httpServer").Values(Dict{
						Id("log"):                Id("log"),
						Id("maxRequestBodySize"): Id("maxRequestBodySize"),
					}),
				}
				for serviceName := range tr.services {
					values[Id("http"+serviceName)] = Id("New"+serviceName).Call(Id("log"), Id("svc"+serviceName))
				}
				sg.Add(values)
			})

			if tr.hasJsonRPC {
				bg.Id("srv").Dot("router").Dot("POST").Call(Lit("/"), Id("srv").Dot("serveBatch"))
			}

			for serviceName, service := range tr.services {

				prefix := service.tags.Value(tagHttpPrefix)

				if service.tags.Contains(tagServerJsonRPC) {

					bg.Id("srv").Dot("router").Dot("POST").Call(Lit(path.Join("/", prefix, service.lcName())), Id("srv").Dot("http"+serviceName).Dot("serveBatch"))

					for _, method := range service.methods {

						if !method.isJsonRPC() {
							continue
						}
						bg.Id("srv").Dot("router").Dot("POST").Call(Lit(method.jsonrpcPath()), Id("srv").Dot("http"+serviceName).Dot("serve"+method.Name))
					}

				}

				if service.tags.Contains(tagServerHTTP) {

					bg.Line()

					for _, method := range service.methods {

						if !method.isHTTP() {
							continue
						}

						if method.tags.Contains(tagHandler) {
							bg.Id("srv").Dot("router").Dot(method.httpMethod()).Call(Lit(method.httpPath()), Qual(method.handlerQual()))
							continue
						}

						bg.Id("srv").Dot("router").Dot(method.httpMethod()).Call(Lit(method.httpPath()), Id("srv").Dot("http"+serviceName).Dot("serve"+method.Name))
					}
				}
			}
			bg.Line()

			// bg.Line().Id("router").Dot("GET").Call(Lit("/liveness"), Func().Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
			// 	Id(_ctx_).Dot("SetStatusCode").Call(Qual(packageFastHttp, "StatusOK")),
			// ))
			//
			// bg.Line().Id("router").Dot("GET").Call(Lit("/readiness"), Func().Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
			// 	Id(_ctx_).Dot("SetStatusCode").Call(Qual(packageFastHttp, "StatusOK")),
			// ))
			//
			// bg.Line().Id("router").Dot("GET").Call(Lit("/metrics"), Qual(packageFastHttpAdapt, "NewFastHTTPHandler").Call(Qual(packagePrometheusHttp, "Handler").Call()))

			bg.Return()
		})
}

func (tr Transport) serveHTTP() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHTTP").Params(Id("address").String(), Id("options").Op("...").Id("Option")).BlockFunc(

		func(bg *Group) {

			bg.Line().Id("srv").Dot("applyOptions").Call(Id("options").Op("..."))

			for serviceName := range tr.services {
				bg.Id("srv").Dot("http" + serviceName).Dot("applyOptions").Call(Id("options").Op("..."))
			}

			bg.Line().Id("srv").Dot("log").Dot("WithField").Call(Lit("address"), Id("address")).Dot("Info").Call(Lit("enable HTTP transport")).Line()

			bg.Line().Id("srv").Dot("srvHttp").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
				Id("ReadTimeout"):        Qual(packageTime, "Second").Op("*").Lit(10),
				Id("Handler"):            Qual(packageCors, "AllowAll").Call().Dot("Handler").Call(Id("srv").Dot("router").Dot("Handler")),
				Id("MaxRequestBodySize"): Id("srv").Dot("maxRequestBodySize"),
			})

			bg.Line().Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvHttp").Dot("ListenAndServe").Call(Id("address")),
				Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve http on ").Op("+").Id("address")),
			).Call()
		},
	)
}

func (tr Transport) serveHTTPS() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHTTPS").Params(List(Id("address"), Id("certFile"), Id("keyFile")).String(), Id("options").Op("...").Id("Option")).BlockFunc(

		func(bg *Group) {

			bg.Line().Id("srv").Dot("applyOptions").Call(Id("options").Op("..."))

			for serviceName := range tr.services {
				bg.Id("srv").Dot("http" + serviceName).Dot("applyOptions").Call(Id("options").Op("..."))
			}

			bg.Line().Id("srv").Dot("log").Dot("WithField").Call(Lit("address"), Id("address")).Dot("Info").Call(Lit("enable HTTP transport")).Line()

			bg.Line().Id("srv").Dot("srvHttp").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
				Id("ReadTimeout"):        Qual(packageTime, "Second").Op("*").Lit(10),
				Id("Handler"):            Qual(packageCors, "AllowAll").Call().Dot("Handler").Call(Id("srv").Dot("router").Dot("Handler")),
				Id("MaxRequestBodySize"): Id("srv").Dot("maxRequestBodySize"),
			})

			bg.Line().Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvHttp").Dot("ListenAndServeTLS").Call(Id("address"), Id("certFile"), Id("keyFile")),
				Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve http on ").Op("+").Id("address")),
			).Call()
		},
	)
}

func (tr Transport) serveHealthFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeHealth").Params(Id("address").String()).Block(

		Line().Id("srv").Dot("srvMetrics").Op("=").Op("&").Qual(packageFastHttp, "Server").Values(Dict{
			Id("ReadTimeout"): Qual(packageTime, "Second").Op("*").Lit(10),
			Id("Handler"): Func().Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
				Id(_ctx_).Dot("SetStatusCode").Call(Qual(packageFastHttp, "StatusOK")),
				List(Id("_"), Id("_")).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("ok")),
			),
		}),

		Line().Go().Func().Params().Block(
			Err().Op(":=").Id("srv").Dot("srvMetrics").Dot("ListenAndServe").Call(Id("address")),
			Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("serve health on ").Op("+").Id("address")),
		).Call(),
	)
}

func (tr Transport) shutdownFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Shutdown").Params().Block(

		Line().If(Id("srv").Op(".").Id("srvHttp").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHttp").Dot("Shutdown").Call(),
		),

		Line().If(Id("srv").Op(".").Id("srvHealth").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHealth").Dot("Shutdown").Call(),
		),

		Line().If(Id("srv").Op(".").Id("srvMetrics").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvMetrics").Dot("Shutdown").Call(),
		),

		Line().If(Id("srv").Op(".").Id("srvPPROF").Op("!=").Id("nil")).Block(
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
