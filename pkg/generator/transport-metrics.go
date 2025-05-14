// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-metrics.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderMetrics(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))

	srcFile.PackageComment(doNotEdit)

	srcFile.ImportAlias(packagePrometheus, "prometheus")

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageFiberAdaptor, "adaptor")
	srcFile.ImportName(packagePrometheusHttp, "promhttp")

	srcFile.Add(Var().Id("VersionGauge").Op("*").Qual(packagePrometheus, "GaugeVec"))
	srcFile.Add(Var().Id("RequestCount").Op("*").Qual(packagePrometheus, "CounterVec"))
	srcFile.Add(Var().Id("RequestCountAll").Op("*").Qual(packagePrometheus, "CounterVec"))
	srcFile.Add(Var().Id("RequestLatency").Op("*").Qual(packagePrometheus, "HistogramVec"))

	srcFile.Add(tr.serveMetricsFunc())

	return srcFile.Save(path.Join(outDir, "metrics.go"))
}

func (tr *Transport) serveMetricsFunc() Code {
	return Func().Params(Id("srv").Op("*").Id("Server")).Id("ServeMetrics").Params(Id("log").Qual(packageZeroLog, "Logger"), Id("path").String(), Id("address").String()).Block(
		Id("srv").Dot("srvMetrics").Op("=").Qual(packageFiber, "New").Call(Qual(packageFiber, "Config").Values(Dict{Id("DisableStartupMessage"): True()})),
		Id("srv").Dot("srvMetrics").Dot("All").Call(Id("path"), Qual(packageFiberAdaptor, "HTTPHandler").Call(Qual(packagePrometheusHttp, "Handler").Call())),
		Go().Func().Params().Block(
			Err().Op(":=").Id("srv").Dot("srvMetrics").Dot("Listen").Call(Id("address")),
			Id("ExitOnError").Call(Id("log"), Err(), Lit("serve metrics on ").Op("+").Id("address")),
		).Call(),
	)
}
