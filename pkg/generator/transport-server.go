// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-server.go at 15.06.2020, 11:38) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderServer(outDir string) (err error) {

	if tr.hasTrace() {
		if err = pkgCopyTo("tracer", outDir); err != nil {
			return
		}
	}
	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageIO, "io")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLogLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packagePrometheus, "prometheus")
	srcFile.ImportName(packagePrometheusAuto, "promauto")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")
	srcFile.ImportName(tr.tags.Value(tagPackageJSON, packageStdJSON), "json")
	if tr.hasTrace() {
		srcFile.ImportName(fmt.Sprintf("%s/tracer", tr.pkgPath(outDir)), "tracer")
	}

	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	}

	srcFile.Line().Add(tr.serverType())
	srcFile.Line().Add(tr.serverNewFunc(outDir))
	srcFile.Line().Add(tr.fiberFunc())
	srcFile.Line().Add(tr.withLogFunc())
	srcFile.Line().Add(tr.serveHealthFunc())
	srcFile.Line().Add(tr.sendResponseFunc())
	srcFile.Line().Add(tr.shutdownFunc())
	if tr.hasTrace() {
		srcFile.Line().Add(tr.withTraceFunc(outDir))
	}
	if tr.hasMetrics() {
		srcFile.Line().Add(tr.withMetricsFunc())
	}
	for _, serviceName := range tr.serviceKeys() {
		srcFile.Line().Add(Func().Params(Id("srv").Op("*").Id("Server")).Id(serviceName).Params().Params(Op("*").Id("http" + serviceName)).Block(
			Return(Id("srv").Dot("http" + serviceName)),
		))
	}

	return srcFile.Save(path.Join(outDir, "server.go"))
}

func (tr *Transport) fiberFunc() Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Fiber").Params().Params(Op("*").Qual(packageFiber, "App")).Block(
		Return(Id("srv").Dot("srvHTTP")),
	)
}

func (tr *Transport) withLogFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithLog").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		for _, serviceName := range tr.serviceKeys() {
			bg.If(Id("srv").Dot("http" + serviceName).Op("!=").Nil()).Block(
				Id("srv").Dot("http" + serviceName).Op("=").Id("srv").Dot(serviceName).Call().Dot("WithLog").Call(),
			)
		}
		bg.Return(Id("srv"))
	})
}

func (tr *Transport) withTraceFunc(outDir string) Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithTrace").
		Params(Id(_ctx_).Qual(packageContext, "Context"), Id("appName").String(), Id("endpoint").String(), Id("attributes").Op("...").Qual(packageAttributeOTEL, "KeyValue")).
		Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		bg.Line().Qual(fmt.Sprintf("%s/tracer", tr.pkgPath(outDir)), "Init").Call(Id(_ctx_), Id("appName"), Id("endpoint"), Id("attributes").Op("..."))
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

func (tr *Transport) withMetricsFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("WithMetrics").Params().Params(Op("*").Id("Server")).BlockFunc(func(bg *Group) {

		bg.If(Id("VersionGauge").Op("==").Nil()).Block(
			Id("VersionGauge").Op("=").Qual(packagePrometheusAuto, "NewGaugeVec").Call(Qual(packagePrometheus, "GaugeOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("count")
					d[Id("Namespace")] = Lit("service")
					d[Id("Subsystem")] = Lit("versions")
					d[Id("Help")] = Lit("Versions of service parts")
				}),
			), Index().String().Values(Lit("part"), Lit("version"), Lit("hostname"))),
		)
		bg.If(Id("RequestCount").Op("==").Nil()).Block(
			Id("RequestCount").Op("=").Qual(packagePrometheusAuto, "NewCounterVec").Call(Qual(packagePrometheus, "CounterOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("count")
					d[Id("Namespace")] = Lit("service")
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Number of requests received")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"))),
		)
		bg.If(Id("RequestCountAll").Op("==").Nil()).Block(
			Id("RequestCountAll").Op("=").Qual(packagePrometheusAuto, "NewCounterVec").Call(Qual(packagePrometheus, "CounterOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("all_count")
					d[Id("Namespace")] = Lit("service")
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Number of all requests received")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"))),
		)
		bg.If(Id("RequestLatency").Op("==").Nil()).Block(
			Id("RequestLatency").Op("=").Qual(packagePrometheusAuto, "NewHistogramVec").Call(Qual(packagePrometheus, "HistogramOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("latency_microseconds")
					d[Id("Namespace")] = Lit("service")
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Total duration of requests in microseconds")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"))),
		)
		bg.List(Id("hostname"), Id("_")).Op(":=").Qual(packageOS, "Hostname").Call()
		bg.Id("VersionGauge").Dot("WithLabelValues").Call(Lit("tg"), Id("VersionTg"), Id("hostname")).Dot("Set").Call(Lit(1))
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

func (tr *Transport) serverType() Code {

	return Type().Id("Server").StructFunc(func(g *Group) {
		g.Id("log").Qual(packageZeroLog, "Logger")
		g.Line().Id("httpAfter").Op("[]").Id("Handler")
		g.Id("httpBefore").Op("[]").Id("Handler")
		g.Line().Id("config").Qual(packageFiber, "Config")
		g.Line().Id("srvHTTP").Op("*").Qual(packageFiber, "App")
		g.Id("srvHealth").Op("*").Qual(packageFiber, "App")
		g.Id("srvMetrics").Op("*").Qual(packageFiber, "App")
		g.Line().Id("reporterCloser").Qual(packageIO, "Closer")
		if tr.hasJsonRPC {
			g.Line().Id("maxBatchSize").Int()
			g.Id("maxParallelBatch").Int().Line()
		}
		for _, serviceName := range tr.serviceKeys() {
			g.Id("http" + serviceName).Op("*").Id("http" + serviceName)
		}
		g.Id("headerHandlers").Map(String()).Id("HeaderHandler")
	})
}

func (tr *Transport) serverNewFunc(outDir string) Code {

	return Func().Id("New").Params(Id("log").Qual(packageZeroLog, "Logger"), Id("options").Op("...").Id("Option")).Params(Id("srv").Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.Line().Id("srv").Op("=").Op("&").Id("Server").Values(DictFunc(func(dict Dict) {

				dict[Id("log")] = Id("log")
				if tr.hasJsonRPC {
					dict[Id("maxBatchSize")] = Id("defaultMaxBatchSize")
					dict[Id("maxParallelBatch")] = Id("defaultMaxParallelBatch")
				}
				dict[Id("headerHandlers")] = Make(Map(String()).Id("HeaderHandler"))
				dict[Id("config")] = Qual(packageFiber, "Config").Values(Dict{
					Id("DisableStartupMessage"): True(),
				})
			},
			))
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			bg.Id("srv").Dot("srvHTTP").Op("=").Qual(packageFiber, "New").Call(Id("srv").Dot("config"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("recoverHandler"))
			if tr.hasTrace() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Qual(fmt.Sprintf("%s/tracer", tr.pkgPath(outDir)), "Middleware").Call())
			}
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("setLogger"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("logLevelHandler"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("headersHandler"))
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			if tr.hasJsonRPC {
				bg.Id("srv").Dot("srvHTTP").Dot("Post").Call(Lit("/"+tr.tags.Value(tagHttpPrefix, "")), Id("srv").Dot("serveBatch"))
			}
			bg.Return()
		})
}

func (tr *Transport) serveHealthFunc() Code {

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

func (tr *Transport) shutdownFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("Shutdown").Params().BlockFunc(func(bg *Group) {
		bg.If(Id("srv").Dot("srvHTTP").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHTTP").Dot("Shutdown").Call(),
		)
		bg.If(Id("srv").Dot("srvHealth").Op("!=").Id("nil")).Block(
			Id("_").Op("=").Id("srv").Dot("srvHealth").Dot("Shutdown").Call(),
		)
		if tr.hasMetrics() {
			bg.If(Id("srv").Dot("srvMetrics").Op("!=").Id("nil")).Block(
				Id("_").Op("=").Id("srv").Dot("srvMetrics").Dot("Shutdown").Call(),
			)
		}
	})
}

func (tr *Transport) sendResponseFunc() Code {
	return Func().Id("sendResponse").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("resp").Interface()).Params(Err().Error()).Block(
		Id(_ctx_).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit("application/json")),
		If(Err().Op("=").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "NewEncoder").Call(Id(_ctx_)).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
			Qual(packageZeroLogLog, "Ctx").Call(Id(_ctx_).Dot("UserContext").Call()).Dot("Error").Call().Dot("Err").Call(Err()).Dot("Str").Call(Lit("body"), String().Call(Id(_ctx_).Dot("Body").Call())).Dot("Msg").Call(Lit("response write error")),
		),
		Return(),
	)
}
