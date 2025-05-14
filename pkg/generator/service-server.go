// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-server.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

func (svc *service) renderServer(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))

	srcFile.Line().Add(svc.serverType())
	srcFile.Line().Add(svc.middlewareSetType())
	srcFile.Line().Add(svc.newServerFunc())
	srcFile.Line().Add(svc.wrapFunc())

	for _, method := range svc.methods {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + svc.Name)).Id(method.Name).Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).Block(
			Return(Id("srv").Dot(method.lccName()).CallFunc(func(cg *Group) {
				for _, arg := range method.Args {

					argCode := Id(arg.Name)

					if types.IsEllipsis(arg.Type) {
						argCode.Op("...")
					}
					cg.Add(argCode)
				}
			})),
		)
	}
	for _, method := range svc.methods {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + svc.Name)).Id("Wrap" + method.Name).Params(Id("m").Id("Middleware" + svc.Name + method.Name)).Block(
			Id("srv").Dot(method.lccName()).Op("=").Id("m").Call(Id("srv").Dot(method.lccName())),
		)
	}
	if svc.tags.Contains(tagTrace) {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + svc.Name)).Id("WithTrace").Params().Block(
			Id("srv").Dot("Wrap").Call(Id("traceMiddleware" + svc.Name)),
		)
	}
	if svc.tags.Contains(tagMetrics) {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + svc.Name)).Id("WithMetrics").Params().Block(
			Id("srv").Dot("Wrap").Call(Id("metricsMiddleware" + svc.Name)),
		)
	}
	if svc.tags.Contains(tagLogger) {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + svc.Name)).Id("WithLog").Params().Block(
			Id("srv").Dot("Wrap").Call(Id("loggerMiddleware" + svc.Name).Call()),
		)
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-server.go"))
}

func (svc *service) wrapFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("server" + svc.Name)).Id("Wrap").Params(Id("m").Id("Middleware" + svc.Name)).BlockFunc(func(bg *Group) {
		bg.Id("srv").Dot("svc").Op("=").Id("m").Call(Id("srv").Dot("svc"))
		for _, method := range svc.methods {
			bg.Id("srv").Dot(method.lccName()).Op("=").Id("srv").Dot("svc").Dot(method.Name)
		}
	})
}

func (svc *service) newServerFunc() Code {

	return Func().Id("newServer" + svc.Name).Params(Id("svc").Qual(svc.pkgPath, svc.Name)).Params(Op("*").Id("server" + svc.Name)).Block(
		Return(Op("&").Id("server" + svc.Name).Values(DictFunc(func(dict Dict) {
			dict[Id("svc")] = Id("svc")
			for _, method := range svc.methods {
				dict[Id(method.lccName())] = Id("svc").Dot(method.Name)
			}
		}))),
	)
}

func (svc *service) serverType() Code {

	return Type().Id("server" + svc.Name).StructFunc(func(sg *Group) {
		sg.Id("svc").Qual(svc.pkgPath, svc.Name)
		for _, method := range svc.methods {
			sg.Id(method.lccName()).Id(svc.Name + method.Name)
		}
	})
}

func (svc *service) middlewareSetType() Code {

	return Type().Id("MiddlewareSet" + svc.Name).InterfaceFunc(func(ig *Group) {

		ig.Id("Wrap").Params(Id("m").Id("Middleware" + svc.Name))
		for _, method := range svc.methods {
			ig.Id("Wrap" + method.Name).Params(Id("m").Id("Middleware" + svc.Name + method.Name))
		}
		ig.Line()
		if svc.tags.IsSet(tagTrace) {
			ig.Id("WithTrace").Params()
		}
		if svc.tags.IsSet(tagMetrics) {
			ig.Id("WithMetrics").Params()
		}
		ig.Id("WithLog").Params()
	})
}
