// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-tracer.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderTracer(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageHttp, "http")
	srcFile.ImportName(packageJaegerlog, "log")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageGotils, "gotils")
	srcFile.ImportName(packageOpenZipkin, "zipkin")
	srcFile.ImportName(packageFastHttp, "fasthttp")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportAlias(packageOpentracing, "otg")
	srcFile.ImportName(packageJaegerConfig, "config")
	srcFile.ImportName(packageJaegerClient, "jaeger")
	srcFile.ImportName(packageJaegerMetrics, "metrics")
	srcFile.ImportAlias(packageZipkinHttp, "httpReporter")
	srcFile.ImportAlias(packageOpenZipkinOpenTracing, "zipkinTracer")

	srcFile.Add(tr.traceJaegerFunc())
	srcFile.Line().Add(tr.traceZipkinFunc())

	srcFile.Line().Add(tr.injectSpanFunc())
	srcFile.Line().Add(tr.extractSpanFunc())

	srcFile.Line().Add(tr.toStringFunc())

	return srcFile.Save(path.Join(outDir, "tracer.go"))
}

func (tr Transport) extractSpanFunc() Code {

	return Func().Id("extractSpan").
		Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id("opName").String(), Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).
		Params(Id("span").Qual(packageOpentracing, "Span")).Block(

		Line().Id("headers").Op(":=").Make(Qual(packageHttp, "Header")),
		Line().Id(_ctx_).Dot("Request").Dot("Header").Dot("VisitAll").Call(Func().Params(Id("key"), Id("value").Op("[]").Byte()).Block(
			Id("headers").Dot("Set").Call(Qual(packageGotils, "B2S").Call(Id("key")), Qual(packageGotils, "B2S").Call(Id("value"))),
		)),

		Line().Var().Id("opts").Op("[]").Qual(packageOpentracing, "StartSpanOption"),
		List(Id("wireContext"), Err()).Op(":=").Qual(packageOpentracing, "GlobalTracer").Call().Dot("Extract").Call(Qual(packageOpentracing, "HTTPHeaders"), Qual(packageOpentracing, "HTTPHeadersCarrier").Call(Id("headers"))),

		Line().If(Err().Op("!=").Nil()).Block(
			Id("log").Dot("WithError").Call(Err()).Dot("Debug").Call(Lit("extract span from HTTP headers")),
		).Else().Block(
			Id("opts").Op("=").Append(Id("opts"), Qual(packageOpentracing, "ChildOf").Call(Id("wireContext"))),
		),
		Line().Id("span").Op("=").Qual(packageOpentracing, "GlobalTracer").Call().Dot("StartSpan").Call(Id("opName"), Id("opts").Op("...")),
		Line().Qual(packageOpentracingExt, "HTTPUrl").Dot("Set").Call(Id("span"), Id(_ctx_).Dot("URI").Call().Dot("String").Call()),
		Qual(packageOpentracingExt, "HTTPMethod").Dot("Set").Call(Id("span"), Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Method").Call())),
		Line().Return(),
	)
}

func (tr Transport) toStringFunc() Code {

	return Func().Id("toString").Params(Id("value").Interface()).String().Block(
		List(Id("data"), Id("_")).Op(":=").Qual(packageJson, "Marshal").Call(Id("value")),
		Return(Qual(packageGotils, "B2S").Call(Id("data"))),
	)
}

func (tr Transport) injectSpanFunc() Code {

	return Func().Id("injectSpan").Params(Id("log").Qual(packageLogrus, "FieldLogger"), Id("span").Qual(packageOpentracing, "Span"), Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(

		Line().Id("headers").Op(":=").Make(Qual(packageHttp, "Header")),
		Line().If(Err().Op(":=").Qual(packageOpentracing, "GlobalTracer").Call().
			Dot("Inject").Call(
			Id("span").Dot("Context").Call(),
			Qual(packageOpentracing, "HTTPHeaders"),
			Qual(packageOpentracing, "HTTPHeadersCarrier").Call(Id("headers")),
		).Op(";").Err().Op("!=").Nil()).Block(
			Id("log").Dot("WithError").Call(Err()).Dot("Debug").Call(Lit("inject span to HTTP headers")),
		),
		Line().For(List(Id("key"), Id("values")).Op(":=").Range().Id("headers")).Block(
			Id(_ctx_).Dot("Response").Dot("Header").Dot("Set").Call(Id("key"), Qual(packageStrings, "Join").Call(Id("values"), Lit(";"))),
		),
	)
}

func (tr Transport) traceJaegerFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("TraceJaeger").Params(Id("serviceName").String()).Params(Op("*").Id("Server")).BlockFunc(func(g *Group) {

		g.Line().List(Id("environment"), Id("_")).Op(":=").Qual(packageOS, "LookupEnv").Call(Lit("ENV"))

		g.Line().List(Id("cfg"), Err()).Op(":=").Qual(packageJaegerConfig, "FromEnv").Call()
		g.Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("jaeger config err"))

		g.Line().If(Id("cfg").Dot("ServiceName").Op("==").Lit("")).Block(
			Id("cfg").Dot("ServiceName").Op("=").Id("environment").Op("+").Id("serviceName"),
		)

		g.Line().Var().Id("trace").Qual(packageOpentracing, "Tracer")
		g.List(Id("trace"), Id("srv").Dot("reporterCloser"), Err()).Op("=").Id("cfg").Dot("NewTracer").Call(
			Qual(packageJaegerConfig, "Logger").Call(Qual(packageJaegerlog, "NullLogger")),
			Qual(packageJaegerConfig, "Metrics").Call(Qual(packageJaegerMetrics, "NullFactory")),
		)

		g.Line().Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("could not create jaeger tracer"))

		g.Line().Qual(packageOpentracing, "SetGlobalTracer").Call(Id("trace"))
		g.Return(Id("srv"))
	})
}

func (tr Transport) traceZipkinFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("TraceZipkin").Params(Id("serviceName").String(), Id("zipkinUrl").String()).Params(Op("*").Id("Server")).BlockFunc(func(g *Group) {

		g.Line().Id("reporter").Op(":=").Qual(packageZipkinHttp, "NewReporter").Call(Id("zipkinUrl"))
		g.Id("srv").Dot("reporterCloser").Op("=").Id("reporter")

		g.Line().List(Id("environment"), Id("envExists")).Op(":=").Qual(packageOS, "LookupEnv").Call(Lit("ENV"))

		g.Line().If(Id("envExists")).Block(Id("serviceName").Op("=").Id("environment").Op("+").Id("serviceName"))

		g.Line().List(Id("endpoint"), Err()).Op(":=").Qual(packageOpenZipkin, "NewEndpoint").Call(Id("serviceName"), Lit(""))
		g.Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("could not create endpoint"))

		g.Line().List(Id("nativeTracer"), Err()).Op(":=").Qual(packageOpenZipkin, "NewTracer").Call(Id("reporter"), Qual(packageOpenZipkin, "WithLocalEndpoint").Call(Id("endpoint")))
		g.Id("ExitOnError").Call(Id("srv").Dot("log"), Err(), Lit("could not create tracer"))

		g.Line().Id("trace").Op(":=").Qual(packageOpenZipkinOpenTracing, "Wrap").Call(Id("nativeTracer"))
		g.Qual(packageOpentracing, "SetGlobalTracer").Call(Id("trace"))

		g.Line().Return(Id("srv"))
	})
}
