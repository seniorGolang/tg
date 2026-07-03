// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-jsonrpc.go at 24.06.2020, 15:31) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageContext, "context")
	srcFile.ImportName(packageErrors, "errors")
	srcFile.ImportName(packageZeroLogLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	srcFile.ImportName(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "json")

	for _, method := range svc.methods {
		if !method.isJsonRPC() {
			continue
		}
		srcFile.Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serve" + method.Name).Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Err().Error()).Block(
			Return().Id("http").Dot("_serveMethod").Call(Id(_ctx_), Lit(method.lcName()), Id("http").Dot(method.lccName())),
		)
		srcFile.Add(svc.rpcMethodFunc(method, outDir))
	}
	srcFile.Add(svc.serveMethodFunc())
	if err = srcFile.Save(path.Join(outDir, svc.lcName()+"-jsonrpc.go")); err != nil {
		return
	}
	return svc.renderBatch(outDir)
}

func (svc *service) renderBatch(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageContext, "context")
	srcFile.ImportName(packageSync, "sync")
	srcFile.ImportName(packageStrings, "strings")
	srcFile.ImportName(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "json")

	srcFile.Add(batchDoFunc(batchTarget{receiver: "http", receiverType: "http" + svc.Name}))
	srcFile.Add(svc.singleBatchFunc())
	srcFile.Add(svc.serveBatchFunc())

	return srcFile.Save(path.Join(outDir, svc.lcName()+"-batch.go"))
}

func (svc *service) rpcMethodFunc(method *method, outDir string) Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id(method.lccName()).
		Params(Id("userCtx").Qual(packageContext, "Context"), Id("ftx").Op("*").Qual(packageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC")).BlockFunc(func(bg *Group) {
		bg.Line()
		bg.Var().Err().Error()
		bg.Var().Id("request").Id(method.requestStructName())
		bg.Var().Id("response").Id(method.responseStructName())
		bg.Line()
		bg.Id("methodCtx").Op(":=").
			Qual(packageZeroLogLog, "Ctx").Call(Id("userCtx")).
			Dot("With").Call().
			Dot("Str").Call(Lit("method"), Lit(method.fullName())).
			Dot("Logger").Call().
			Dot("WithContext").Call(Id("userCtx"))
		bg.Line()
		bg.If(Id("requestBase").Dot("Params").Op("!=").Nil()).Block(
			If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			}),
		)
		bg.If(Id("requestBase").Dot("Version").Op("!=").Id("Version")).BlockFunc(func(ig *Group) {
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version"), Nil()))
		})
		if method.hasFiberRequest() {
			bg.If(Id("ftx").Op("!=").Nil()).Block(
				Add(method.httpArgHeaders("ftx", func(arg, header string) *Statement {
					return Line().If(Err().Op("!=").Nil()).Block(
						Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
					)
				})),
				Add(method.httpCookies("ftx", func(arg, header string) *Statement {
					return Line().If(Err().Op("!=").Nil()).Block(
						Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
					)
				})),
			)
		}
		bg.ListFunc(func(lg *Group) {
			for _, ret := range method.resultsWithoutError() {
				lg.Id("response").Dot(utils.ToCamel(ret.Name))
			}
			lg.Err()
		}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
			cg.Id("methodCtx")
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
			ig.Id("code").Op(":=").Id("internalError")
			ig.If(List(Id("errCoder"), Id("ok")).Op(":=").Err().Op(".").Call(Id("withErrorCode")).Op(";").Id("ok")).Block(
				Id("code").Op("=").Id("errCoder").Dot("Code").Call(),
			)
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("code"), Err().Dot("Error").Call(), Err()))
		})
		bg.Id("responseBase").Op("=").Op("&").Id("baseJsonRPC").Values(Dict{
			Id("Version"): Id("Version"),
			Id("ID"):      Id("requestBase").Dot("ID"),
		})

		resp := Id("response")
		if len(method.resultsWithoutError()) == 1 && method.tags.IsSet(tagHttpEnableInlineSingle) {
			resp = Id("response").Dot(utils.ToCamel(method.resultsWithoutError()[0].Name))
		}
		bg.If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Marshal").Call(resp).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
		})
		if len(method.retCookieMap()) > 0 {
			bg.If(Id("ftx").Op("!=").Nil()).BlockFunc(func(cg *Group) {
				for retName := range method.retCookieMap() {
					if ret := method.resultByName(retName); ret != nil {
						cg.If(List(Id("rCookie"), Id("ok")).Op(":=").
							Qual(packageReflect, "ValueOf").Call(Id("response").Dot(utils.ToCamel(retName))).Dot("Interface").Call().
							Op(".").Call(Id("cookieType"))).Op(";").Id("ok").Op("&&").Id("response").Dot(utils.ToCamel(retName)).Op("!=").Nil().Block(
							Id("ftx").Dot("Cookie").Call(Id("rCookie").Dot("Cookie").Call()),
						)
					}
				}
			})
		}
		if method.hasFiberRetHeaders() {
			bg.If(Id("ftx").Op("!=").Nil()).Block(
				Add(method.httpRetHeaders("ftx")),
			)
		}
		bg.Return()
	})
}

func (svc *service) serveMethodFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("_serveMethod").
		ParamsFunc(func(pg *Group) {
			pg.Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")
			pg.Id("methodName").String()
			pg.Id("methodHandler").Id("methodJsonRPC")
		}).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("methodHTTP").Op(":=").Id(_ctx_).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(packageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
			})
			bg.Var().Id("request").Id("baseJsonRPC")
			bg.Var().Id("response").Op("*").Id("baseJsonRPC")
			bg.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			})
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))

			bg.If(Id("method").Op("!=").Lit("").Op("&&").Id("method").Op("!=").Id("methodName")).BlockFunc(func(ig *Group) {
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method ").Op("+").Id("methodNameOrigin"), Nil()))
			})
			bg.Id("response").Op("=").Id("methodHandler").Call(Id(_ctx_).Dot("UserContext").Call(), Id(_ctx_), Id("request"))
			bg.If(Id("response").Op("!=").Nil()).Block(
				Return().Id("sendResponse").Call(Id(_ctx_), Id("response")),
			)
			bg.Return()
		})
}

func (svc *service) singleBatchFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id("doSingleBatch").
		Params(Id("userCtx").Qual(packageContext, "Context"), Id("ftx").Op("*").Qual(packageFiber, "Ctx"), Id("request").Id("baseJsonRPC")).Params(Id("response").Op("*").Id("baseJsonRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))
			bg.Switch(Id("method")).BlockFunc(
				func(sg *Group) {
					for _, method := range svc.methods {
						if !method.isJsonRPC() {
							continue
						}
						sg.Case(Lit(method.lcName())).Block(
							Return(Id("http").Dot(utils.ToLowerCamel(method.Name)).Call(Id("userCtx"), Id("ftx"), Id("request"))),
						)
					}
					sg.Default().BlockFunc(func(dg *Group) {
						dg.Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil()))
					})
				})
		})
}

func (svc *service) serveBatchFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serveBatch").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Var().Id("single").Bool()
			bg.Var().Id("requests").Op("[]").Id("baseJsonRPC")
			bg.Id("methodHTTP").Op(":=").Id(_ctx_).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(packageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				ig.Return()
			})
			bg.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Var().Id("request").Id("baseJsonRPC")
				ig.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
				})
				ig.Id("single").Op("=").True()
				ig.Id("requests").Op("=").Append(Id("requests"), Id("request"))
			})
			bg.If(Id("single")).Block(
				Return(Id("sendResponse").Call(Id(_ctx_), Id("http").Dot("doSingleBatch").
					Call(Id(_ctx_).Dot("UserContext").Call(), Id(_ctx_), Id("requests").Op("[").Lit(0).Op("]")),
				)),
			)
			bg.Return(Id("sendResponse").Call(Id(_ctx_), Id("http").Dot("doBatch").
				Call(Id(_ctx_), Id("requests")),
			))
		})
}
