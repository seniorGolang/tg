// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (service-metrics.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (svc *service) renderMetrics(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), "code", srcFile)

	srcFile.ImportName(packageGoKitMetrics, "metrics")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("metrics"+svc.Name).Struct(
		Id(_next_).Qual(svc.pkgPath, svc.Name),
		Id("requestCount").Qual(packageGoKitMetrics, "Counter"),
		Id("requestCountAll").Qual(packageGoKitMetrics, "Counter"),
		Id("requestLatency").Qual(packageGoKitMetrics, "Histogram"),
	)

	srcFile.Line().Add(svc.metricsMiddleware())

	for _, method := range svc.methods {
		srcFile.Line().Func().Params(Id("m").Id("metrics" + svc.Name)).Id(method.Name).Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).BlockFunc(svc.metricFuncBody(method))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-metrics.go"))
}

func (svc *service) metricsMiddleware() Code {

	return Func().Id("metricsMiddleware" + svc.Name).Params(Id(_next_).Qual(svc.pkgPath, svc.Name)).Params(Qual(svc.pkgPath, svc.Name)).
		BlockFunc(func(g *Group) {
			g.Return(Op("&").Id("metrics" + svc.Name).Values(
				Dict{
					Id(_next_):            Id(_next_),
					Id("requestCount"):    Id("RequestCount").Op(".").Id("With").Call(Lit("service"), Lit(svc.Name)),
					Id("requestCountAll"): Id("RequestCountAll").Op(".").Id("With").Call(Lit("service"), Lit(svc.Name)),
					Id("requestLatency"):  Id("RequestLatency").Op(".").Id("With").Call(Lit("service"), Lit(svc.Name)),
				},
			))
		})
}

func (svc *service) metricFuncBody(method *method) func(g *Group) {

	return func(g *Group) {

		g.Line().Defer().Func().Params(Id("begin").Qual(packageTime, "Time")).Block(
			Id("m").Dot("requestLatency").Dot("With").Call(
				Lit("method"), Lit(method.lccName()),
				Lit("success"), Qual(packageFmt, "Sprint").Call(Err().Op("==").Nil())).
				Dot("Observe").Call(Qual(packageTime, "Since").Call(Id("begin")).Dot("Seconds").Call()),
		).Call(Qual(packageTime, "Now").Call())

		g.Line().Defer().Id("m").Dot("requestCount").Dot("With").Call(
			Lit("method"), Lit(method.lccName()),
			Lit("success"), Qual(packageFmt, "Sprint").Call(Err().Op("==").Nil())).
			Dot("Add").Call(Lit(1))

		g.Line().Id("m").Dot("requestCountAll").Dot("With").Call(
			Lit("method"), Lit(method.lccName())).
			Dot("Add").Call(Lit(1))

		g.Line().Return().Id("m").Dot(_next_).Dot(method.Name).Call(paramNames(method.Args))
	}
}
