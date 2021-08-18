// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (service-rest.go at 23.06.2020, 23:36) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/vetcher/go-astra/types"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderREST(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	srcFile.ImportName(packageOpentracing, "opentracing")

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

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id(method.lccName()).Params(Id(_ctx_).Qual(packageContext, "Context"), Id("request").Id(method.requestStructName())).
		Params(Id("response").Id(method.responseStructName()), Err().Error()).
		BlockFunc(func(bg *Group) {
			if svc.tags.IsSet(tagTrace) {
				bg.Id("span").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id(_ctx_))
			}
			bg.ListFunc(func(lg *Group) {
				for _, ret := range method.resultsWithoutError() {
					lg.Id("response").Dot(utils.ToCamel(ret.Name))
				}
				lg.Err()

			}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
				cg.Id(_ctx_)
				for _, arg := range method.argsWithoutContext() {
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
		})
}

func (svc *service) httpServeMethodFunc(method *method) Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serve" + method.Name).Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).
		Params(Err().Error()).BlockFunc(func(bg *Group) {

		if svc.tags.IsSet(tagTrace) {
			bg.Id("span").Op(":=").Id("extractSpan").Call(
				Id("http").Dot("log"),
				Qual(packageFmt, "Sprintf").Call(Lit("request:%s"), Id(_ctx_).Dot("Path").Call()),
				Id(_ctx_),
			)
			bg.Defer().Id("injectSpan").Call(Id("http").Dot("log"), Id("span"), Id(_ctx_))
			bg.Defer().Id("span").Dot("Finish").Call()
		}

		bg.If(Id("value").Op(":=").Id(_ctx_).Dot("Context").Call().Dot("Value").Call(Id("CtxCancelRequest")).Op(";").Id("value").Op("!=").Nil()).BlockFunc(func(ig *Group) {
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request canceled"))
			}
			ig.Return()
		})
		bg.Var().Id("request").Id(method.requestStructName())
		if successCode := method.tags.ValueInt(tagHttpSuccess, 0); successCode != 0 {
			bg.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Lit(successCode))
		}
		if len(method.arguments()) != 0 {
			bg.If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("Request").Call().Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusBadRequest"))
				ig.Id(_ctx_).Dot("WriteString").Call(Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}
		bg.Add(method.urlArgs(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("path arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Lit("path arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}))

		bg.Add(method.urlParams(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("url arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Lit("url arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}))
		bg.Add(method.httpArgHeaders(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}))
		bg.Add(method.httpCookies(func(arg, header string) *Statement {
			return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}))
		for uploadVar, uploadKey := range method.uploadVarsMap() {
			bg.If(List(Id("request").Dot(utils.ToCamel(uploadVar)), Err()).Op("=").Id("uploadFile").Call(Id(_ctx_), Lit(uploadKey)).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("upload file '"+uploadVar+"' error: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusBadRequest"))
				ig.Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Lit("upload file '"+uploadVar+"' error: ").Op("+").Err().Dot("Error").Call())
				ig.Return()
			})
		}
		if responseMethod := method.tags.Value(tagHttpResponse, ""); responseMethod != "" {
			bg.Add(toID(responseMethod).Call(Id(_ctx_), Id("http").Dot("base"), Err(), callParamNames("request", method.argsWithoutContext())))
		} else {
			bg.Var().Id("result").Interface()
			bg.Var().Id("response").Id(method.responseStructName())
			if svc.tags.IsSet(tagTrace) {
				bg.Id("methodContext").Op(":=").Qual(packageOpentracing, "ContextWithSpan").Call(Id(_ctx_).Dot("Context").Call(), Id("span"))
			} else {
				bg.Id("methodContext").Op(":=").Id(_ctx_).Dot("Context").Call()
			}
			bg.List(Id("response"), Err()).Op("=").Id("http").Dot(method.lccName()).Call(Id("methodContext"), Id("request"))
			bg.Id("result").Op("=").Id("response")

			ex := Line()
			if len(method.retCookieMap()) > 0 {
				for retName := range method.retCookieMap() {
					if ret := method.resultByName(retName); ret != nil {
						ex.If(List(Id("rCookie"), Id("ok")).Op(":=").
							Qual(packageReflect, "ValueOf").Call(Id("response").Dot(utils.ToCamel(retName))).Dot("Interface").Call().
							Op(".").Call(Id("cookieType"))).Op(";").Id("ok").Op("&&").Id("response").Dot(utils.ToCamel(retName)).Op("!=").Nil().Block(
							Id(_ctx_).Dot("Response").Dot("Header").Dot("SetCookie").Call(Id("rCookie").Dot("Cookie").Call()),
						)
					}
				}
			}
			ex.Add(method.httpRetHeaders())
			if len(*ex) > 1 {
				bg.If(Err().Op("==").Nil()).Block(ex)
			}
			bg.If(Err().Op("!=").Nil()).Block(
				Id("result").Op("=").Err(),
				If(List(Id("errCoder"), Id("ok")).Op(":=").Err().Op(".").Call(Id("withErrorCode")).Op(";").Id("ok")).Block(
					Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Id("errCoder").Dot("Code").Call()),
				).Else().Block(
					Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusInternalServerError")),
				),
			)
			bg.Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("result"))
		}
		bg.Return()
	})
}

func toID(str string) *Statement {
	if tokens := strings.Split(str, ":"); len(tokens) == 2 {
		return Qual(tokens[0], tokens[1])
	}
	return Id(str)
}
