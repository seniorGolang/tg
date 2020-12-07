// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (service-logger.go at 19.06.2020, 16:10) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/pkg/tags"
	"github.com/seniorGolang/tg/pkg/utils"
)

func (svc *service) renderLogger(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), "code", srcFile)

	srcFile.ImportName(packageViewer, "viewer")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageGoKitMetrics, "metrics")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("logger"+svc.Name).Struct(
		Id(_next_).Qual(svc.pkgPath, svc.Name),
		Id("log").Qual(packageLogrus, "FieldLogger"),
	)

	srcFile.Line().Add(svc.loggerMiddleware())

	for _, method := range svc.methods {
		srcFile.Line().Func().Params(Id("m").Id("logger" + svc.Name)).Id(method.Name).Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).BlockFunc(svc.loggerFuncBody(method))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-logger.go"))
}

func (svc *service) loggerMiddleware() Code {

	return Func().Id("loggerMiddleware" + svc.Name).Params(Id("log").Qual(packageLogrus, "FieldLogger")).Params(Id("Middleware" + svc.Name)).Block(
		Return(Func().Params(Id(_next_).Qual(svc.pkgPath, svc.Name)).Params(Qual(svc.pkgPath, svc.Name)).Block(
			Return(Op("&").Id("logger" + svc.Name).Values(Dict{
				Id("log"):  Id("log"),
				Id(_next_): Id(_next_),
			})),
		)),
	)
}

func (svc *service) loggerFuncBody(method *method) func(g *Group) {

	return func(g *Group) {

		g.Defer().Func().Params(Id("begin").Qual(packageTime, "Time")).BlockFunc(func(g *Group) {

			g.Id("fields").Op(":=").Qual(packageLogrus, "Fields").Values(DictFunc(func(d Dict) {

				// for _, header := range method.argHeaderMap() {
				// 	d[Lit(header)] = Id(_ctx_).Op(".").Id("Value").Call(Lit(header))
				// }

				d[Lit("service")] = Lit(svc.Name)
				d[Lit("method")] = Lit(method.lccName())

				skipFields := strings.Split(tags.ParseTags(method.Docs).Value("log-skip"), ",")
				params := method.argsWithoutContext()
				params = removeSkippedFields(params, skipFields)

				d[Lit("request")] = Qual(packageViewer, "Sprintf").Call(Lit("%+v"), Id(method.requestStructName()).Values(utils.DictByNormalVariables(params, params)))

				printResult := true
				for _, field := range skipFields {
					if strings.TrimSpace(field) == "response" {
						printResult = false
						break
					}
				}
				returns := method.resultsWithoutError()

				if printResult {
					d[Lit("response")] = Qual(packageViewer, "Sprintf").Call(Lit("%+v"), Id(method.responseStructName()).Values(utils.DictByNormalVariables(returns, returns)))
				}

				d[Lit("took")] = Qual(packageTime, "Since").Call(Id("begin"))
			}))

			g.If(Id(_ctx_).Dot("Value").Call(Id("headerRequestID")).Op("!=").Nil()).Block(
				Id("fields").Op("[").Lit("requestID").Op("]").Op("=").Id(_ctx_).Dot("Value").Call(Id("headerRequestID")),
			)

			g.If(Id("err").Op("!=").Id("nil")).BlockFunc(func(g *Group) {

				g.Id("m").Dot("log").Dot("WithError").Call(Err()).Dot("WithFields").Call(Id("fields")).Dot("Info").Call(Lit(fmt.Sprintf("call %s", method.lccName())))
				g.Return()
			})

			g.Id("m").Dot("log").Dot("WithFields").Call(Id("fields")).Dot("Info").Call(Lit(fmt.Sprintf("call %s", method.lccName())))

		}).Call(Qual(packageTime, "Now").Call())
		g.Return().Id("m").Dot(_next_).Dot(method.Name).Call(paramNames(method.Args))
	}
}
