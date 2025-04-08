// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-metrics.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (svc *service) renderMetrics(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportName(packagePrometheus, "metrics")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("metrics" + svc.Name).Struct(
		Id(_next_).Qual(svc.pkgPath, svc.Name),
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
					Id(_next_): Id(_next_),
				},
			))
		})
}

func (svc *service) metricFuncBody(method *method) func(g *Group) {

	return func(g *Group) {

		errCodeAssignment := Id("errCode").Op("=")

		if method.isHTTP() {
			errCodeAssignment.Qual(packageFiber, "StatusInternalServerError")
		} else {
			errCodeAssignment.Id("internalError")
		}

		g.Line().Defer().Func().Params(Id("_begin").Qual(packageTime, "Time")).Block(
			Var().Defs(
				Id("success").Op("=").True(),
				Id("errCode").Int(),
			),
			If(Err().Op("!=").Nil()).Block(
				Id("success").Op("=").False(),
				errCodeAssignment,
				List(Id("ec"), Id("ok")).Op(":=").Err().Assert(Id("withErrorCode")),
				If(Id("ok")).Block(
					Id("errCode").Op("=").Id("ec").Dot("Code").Call(),
				),
			),
			Id("RequestCount").Dot("WithLabelValues").Call(
				Lit(method.svc.lccName()),
				Lit(method.lccName()),
				Qual("strconv", "FormatBool").Call(Id("success")),
				Qual("strconv", "Itoa").Call(Id("errCode"))).
				Dot("Add").Call(Lit(1)),
			Id("RequestCountAll").Dot("WithLabelValues").Call(
				Lit(method.svc.lccName()),
				Lit(method.lccName()),
				Qual("strconv", "FormatBool").Call(Id("success")),
				Qual("strconv", "Itoa").Call(Id("errCode"))).
				Dot("Add").Call(Lit(1)),
			Id("RequestLatency").Dot("WithLabelValues").Call(
				Lit(method.svc.lccName()),
				Lit(method.lccName()),
				Qual("strconv", "FormatBool").Call(Id("success")),
				Qual("strconv", "Itoa").Call(Id("errCode"))).
				Dot("Observe").Call(Qual(packageTime, "Since").Call(Id("_begin")).Dot("Seconds").Call()),
		).Call(Qual(packageTime, "Now").Call())

		g.Line().Return().Id("m").Dot(_next_).Dot(method.Name).Call(paramNames(method.Args))
	}
}
