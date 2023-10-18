// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-rest.go at 23.06.2020, 23:36) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderREST(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "json")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	srcFile.ImportAlias(packageOpentracing, "otg")

	for _, method := range svc.methods {
		if !method.isHTTP() {
			continue
		}
		srcFile.Add(svc.httpMethodFunc(method))
		srcFile.Add(svc.httpServeMethodFunc(method))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-rest.go"))
}

func (svc *service) httpMethodFunc(method *method) Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id(method.lccName()).Params(
		ListFunc(func(lg *Group) {
			lg.Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")
			if len(method.arguments()) != 0 {
				lg.Id("request").Id(method.requestStructName())
			}
		})).
		Params(ListFunc(func(lg *Group) {
			if len(method.results()) != 0 {
				lg.Id("response").Id(method.responseStructName())
			}
			lg.Err().Error()
		})).
		BlockFunc(func(bg *Group) {
			bg.Line()
			if svc.tags.IsSet(tagTrace) {
				bg.Id("span").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id(_ctx_).Dot("UserContext").Call())
				bg.Id("span").Dot("SetTag").Call(Lit("method"), Lit(method.lccName())).Line()
			}
			bg.ListFunc(func(lg *Group) {
				for _, ret := range method.resultFieldsWithoutError() {
					lg.Id("response").Dot(utils.ToCamel(ret.Name))
				}
				lg.Err()

			}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
				cg.Id(_ctx_)
				for _, arg := range method.argsFieldsWithoutContext() {
					argCode := Id("request").Dot(utils.ToCamel(arg.Name))
					if types.IsEllipsis(arg.Type) {
						argCode.Op("...")
					}
					cg.Add(argCode)
				}
			})
			bg.If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("http").Dot("errorHandler").Op("!=").Nil()).Block(
					Err().Op("=").Id("http").Dot("errorHandler").Call(Err()),
				)
				if svc.tags.IsSet(tagTrace) {
					ig.Id("errData").Op(":=").Id("toString").Call(Err())
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Err().Dot("Error").Call())
					ig.If(Id("errData").Op("!=").Lit("{}")).Block(
						Id("span").Dot("SetTag").Call(Lit("errData"), Id("errData")),
					)
				}
			})
			bg.Return()
		}).Line()
}

func (svc *service) httpServeMethodFunc(method *method) Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serve" + method.Name).Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).
		Params(Err().Error()).BlockFunc(func(bg *Group) {

		bg.Line()
		if svc.tags.IsSet(tagTrace) {
			bg.Id("span").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id(_ctx_).Dot("UserContext").Call())
			bg.Id("span").Dot("SetTag").Call(Lit("method"), Lit(method.lccName())).Line()
		}

		if successCode := method.tags.ValueInt(tagHttpSuccess, 0); successCode != 0 {
			bg.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Lit(successCode))
		}
		if len(method.arguments()) != 0 {
			bg.Var().Id("request").Id(method.requestStructName())
			bg.If(Err().Op("=").Id(_ctx_).Dot("BodyParser").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusBadRequest"))
				ig.List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}
		bg.Add(method.urlArgs(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id(_ctx_).Dot("Status").Call(Qual(packageFiber, "StatusBadRequest"))
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("path arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Lit("path arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
			})
		}))
		bg.Add(method.urlParams(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id(_ctx_).Dot("Status").Call(Qual(packageFiber, "StatusBadRequest"))
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("url arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Lit("url arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
			})
		}))
		bg.Add(method.httpArgHeaders(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id(_ctx_).Dot("Status").Call(Qual(packageFiber, "StatusBadRequest"))
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
			})
		}))
		bg.Add(method.httpCookies(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id(_ctx_).Dot("Status").Call(Qual(packageFiber, "StatusBadRequest"))
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
			})
		}))
		if responseMethod := method.tags.Value(tagHttpResponse, ""); responseMethod != "" {
			bg.Return().Add(toID(responseMethod).Call(Id(_ctx_), Id("http").Dot("svc"), callParamNames("request", method.argsWithoutContext())))
		} else {
			if len(method.results()) == 0 {
				bg.If().List(Err()).Op("=").Id("http").Dot(method.lccName()).Call(ListFunc(func(lg *Group) {
					lg.Id(_ctx_)
					if len(method.arguments()) != 0 {
						lg.Id("request")
					}
				})).Op(";").Err().Op("==").Nil().BlockFunc(func(bf *Group) {
					bf.Return().Nil()
				})
			} else {
				bg.Var().Id("response").Id(method.responseStructName())
				bg.If().List(Id("response"), Err()).Op("=").Id("http").Dot(method.lccName()).Call(Id(_ctx_), Id("request")).Op(";").Err().Op("==").Nil().BlockFunc(func(bf *Group) {
					ex := Line()
					if len(method.retCookieMap()) > 0 {
						for retName := range method.retCookieMap() {
							if ret := method.resultByName(retName); ret != nil {
								ex.If(List(Id("rCookie"), Id("ok")).Op(":=").
									Qual(packageReflect, "ValueOf").Call(Id("response").Dot(utils.ToCamel(retName))).Dot("Interface").Call().
									Op(".").Call(Id("cookieType"))).Op(";").Id("ok").Op("&&").Id("response").Dot(utils.ToCamel(retName)).Op("!=").Nil().Block(
									Id(_ctx_).Dot("Cookie").Call(Id("rCookie").Dot("Cookie").Call()),
								)
							}
						}
					}
					ex.Add(method.httpRetHeaders())
					if len(*ex) > 2 {
						bf.If(Err().Op("==").Nil()).Block(ex)
					}
					bf.Var().Id("iResponse").Interface().Op("=").Id("response")
					bf.If(List(Id("redirect"), Id("ok")).Op(":=").Id("iResponse").Op(".").Call(Id("withRedirect")).Op(";").Id("ok")).Block(
						Return().Id(_ctx_).Dot("Redirect").Call(Id("redirect").Dot("RedirectTo").Call()),
					)
					bf.Return().Id("sendResponse").Call(Id(_ctx_), Id("response"))
				})

			}
			bg.If(List(Id("errCoder"), Id("ok")).Op(":=").Err().Op(".").Call(Id("withErrorCode")).Op(";").Id("ok")).Block(
				Id(_ctx_).Dot("Status").Call(Id("errCoder").Dot("Code").Call()),
			).Else().Block(
				Id(_ctx_).Dot("Status").Call(Qual(packageFiber, "StatusInternalServerError")),
			)
			bg.Return().Id("sendResponse").Call(Id(_ctx_), Err())
		}
	})
}

func toID(str string) *Statement {
	if tokens := strings.Split(str, ":"); len(tokens) == 2 {
		return Qual(tokens[0], tokens[1])
	}
	return Id(str)
}
