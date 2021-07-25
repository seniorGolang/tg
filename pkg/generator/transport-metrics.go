// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-metrics.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderMetrics(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))

	srcFile.PackageComment(doNotEdit)

	srcFile.ImportAlias(packageKitPrometheus, "kitPrometheus")
	srcFile.ImportAlias(packageStdPrometheus, "stdPrometheus")

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageGoKitMetrics, "metrics")
	srcFile.ImportName(packageFiberAdaptor, "adaptor")
	srcFile.ImportName(packageGoKitEndpoint, "endpoint")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")

	srcFile.Var().Id("srvMetrics").Op("*").Qual(packageFiber, "App")

	srcFile.Add(prometheusCounterRequestCount())
	srcFile.Add(prometheusCounterRequestCountAll())
	srcFile.Add(prometheusSummaryRequestCount())

	srcFile.Add(tr.serveMetricsFunc())

	return srcFile.Save(path.Join(outDir, "metrics.go"))
}

func (tr Transport) serveMetricsFunc() Code {
	return Func().Id("ServeMetrics").Params(Id("log").Qual(packageZeroLog, "Logger"), Id("address").String()).Block(
		// fiber.Config{DisableStartupMessage: true}`
		Id("srvMetrics").Op("=").Qual(packageFiber, "New").Call(Qual(packageFiber, "Config").Values(Dict{Id("DisableStartupMessage"): True()})),
		Id("srvMetrics").Dot("All").Call(Lit("/"), Qual(packageFiberAdaptor, "HTTPHandler").Call(Qual(packagePrometheusHttp, "Handler").Call())),
		Go().Func().Params().Block(
			Err().Op(":=").Id("srvMetrics").Dot("Listen").Call(Id("address")),
			Id("ExitOnError").Call(Id("log"), Err(), Lit("serve metrics on ").Op("+").Id("address")),
		).Call(),
	)
}

func prometheusCounterRequestCount() (code *Statement) {

	return Var().Id("RequestCount").Op("=").Qual(packageKitPrometheus, "NewCounterFrom").Call(Qual(packageStdPrometheus, "CounterOpts").Values(
		DictFunc(func(d Dict) {
			d[Id("Name")] = Lit("count")
			d[Id("Namespace")] = Lit("service")
			d[Id("Subsystem")] = Lit("requests")
			d[Id("Help")] = Lit("Number of requests received")
		}),
	), Index().String().Values(Lit("method"), Lit("service"), Lit("success")))
}

func prometheusCounterRequestCountAll() (code *Statement) {

	return Var().Id("RequestCountAll").Op("=").Qual(packageKitPrometheus, "NewCounterFrom").Call(Qual(packageStdPrometheus, "CounterOpts").Values(
		DictFunc(func(d Dict) {
			d[Id("Name")] = Lit("all_count")
			d[Id("Namespace")] = Lit("service")
			d[Id("Subsystem")] = Lit("requests")
			d[Id("Help")] = Lit("Number of all requests received")
		}),
	), Index().String().Values(Lit("method"), Lit("service")))
}

func prometheusSummaryRequestCount() (code *Statement) {

	return Var().Id("RequestLatency").Op("=").Qual(packageKitPrometheus, "NewSummaryFrom").Call(Qual(packageStdPrometheus, "SummaryOpts").Values(
		DictFunc(func(d Dict) {
			d[Id("Name")] = Lit("latency_microseconds")
			d[Id("Namespace")] = Lit("service")
			d[Id("Subsystem")] = Lit("requests")
			d[Id("Help")] = Lit("Total duration of requests in microseconds")
		}),
	), Index().String().Values(Lit("method"), Lit("service"), Lit("success")))
}
