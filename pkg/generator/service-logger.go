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

	"github.com/seniorGolang/tg/v2/pkg/tags"
	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderLogger(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportName(packageViewer, "viewer")
	srcFile.ImportName(packageZeroLogLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageGoKitMetrics, "metrics")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Type().Id("logger" + svc.Name).Struct(
		Id(_next_).Qual(svc.pkgPath, svc.Name),
	)

	srcFile.Line().Add(svc.loggerMiddleware())

	for _, method := range svc.methods {
		srcFile.Line().Func().Params(Id("m").Id("logger" + svc.Name)).Id(method.Name).Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).BlockFunc(svc.loggerFuncBody(method))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-logger.go"))
}

func (svc *service) loggerMiddleware() Code {

	return Func().Id("loggerMiddleware" + svc.Name).Params().Params(Id("Middleware" + svc.Name)).Block(
		Return(Func().Params(Id(_next_).Qual(svc.pkgPath, svc.Name)).Params(Qual(svc.pkgPath, svc.Name)).Block(
			Return(Op("&").Id("logger" + svc.Name).Values(Dict{
				Id(_next_): Id(_next_),
			})),
		)),
	)
}

func (svc *service) loggerFuncBody(method *method) func(g *Group) {

	return func(g *Group) {
		g.Id("logger").Op(":=").Qual(packageZeroLogLog, "Ctx").Call(Id("ctx")).Dot("With").Call().
			Dot("Str").Call(Lit("service"), Lit(svc.Name)).
			Dot("Str").Call(Lit("method"), Lit(method.lccName())).
			Dot("Logger").Call()
		g.Defer().Func().Params(Id("begin").Qual(packageTime, "Time")).BlockFunc(func(g *Group) {
			g.Id("fields").Op(":=").Map(String()).Interface().Values(DictFunc(func(d Dict) {

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
				d[Lit("took")] = Qual(packageTime, "Since").Call(Id("begin")).Dot("String").Call()
			}))
			g.If(Id("err").Op("!=").Id("nil")).BlockFunc(func(g *Group) {
				g.Id("logger").Dot("Error").Call().Dot("Err").Call(Err()).Dot("Fields").Call(Id("fields")).Dot("Msg").Call(Lit(fmt.Sprintf("call %s", method.lccName())))
				g.Return()
			})
			g.Id("logger").Dot("Info").Call().Dot("Fields").Call(Id("fields")).Dot("Msg").Call(Lit(fmt.Sprintf("call %s", method.lccName())))

		}).Call(Qual(packageTime, "Now").Call())
		g.Return().Id("m").Dot(_next_).Dot(method.Name).Call(paramNames(method.Args))
	}
}
