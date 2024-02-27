// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-jsonrpc.go at 24.06.2020, 15:31) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageErrors, "errors")
	srcFile.ImportName(packageZeroLogLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportAlias(packageOpentracing, "otg")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	srcFile.ImportAlias(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "json")

	for _, method := range svc.methods {
		if !method.isJsonRPC() {
			continue
		}
		srcFile.Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serve" + method.Name).Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Err().Error()).Block(
			Return().Id("http").Dot("serveMethod").Call(Id(_ctx_), Lit(method.lcName()), Id("http").Dot(method.lccName())),
		)
		srcFile.Add(svc.rpcMethodFunc(method))
	}
	srcFile.Add(svc.serveMethodFunc())
	srcFile.Add(svc.batchFunc())
	srcFile.Add(svc.serveBatchFunc())
	srcFile.Add(svc.singleBatchFunc())
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-jsonrpc.go"))
}

func (svc *service) rpcMethodFunc(method *method) Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id(method.lccName()).
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC")).BlockFunc(func(bg *Group) {
		bg.Line()
		bg.Var().Err().Error()
		bg.Var().Id("request").Id(method.requestStructName())
		bg.Line()
		bg.Id("methodCtx").Op(":=").Id(_ctx_).Dot("UserContext").Call()
		bg.Id("methodCtx").Op("=").
			Qual(packageZeroLogLog, "Ctx").Call(Id("methodCtx")).
			Dot("With").Call().
			Dot("Str").Call(Lit("method"), Lit(method.fullName())).
			Dot("Logger").Call().
			Dot("WithContext").Call(Id("methodCtx"))
		if svc.tags.IsSet(tagTrace) {
			bg.Id("span").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id("methodCtx"))
			bg.Id("span").Dot("SetTag").Call(Lit("method"), Lit(method.lccName())).Line()
		}
		bg.If(Id("requestBase").Dot("Params").Op("!=").Nil()).Block(
			If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			}),
		)
		bg.If(Id("requestBase").Dot("Version").Op("!=").Id("Version")).BlockFunc(func(ig *Group) {
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version"))
			}
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version"), Nil()))
		})
		bg.Add(method.httpArgHeaders(func(arg, header string) *Statement {
			if svc.tags.IsSet(tagTrace) {
				return Line().Id("span").Dot("SetTag").Call(Lit(header), Id("_"+arg)).
					Line().If(Err().Op("!=").Nil()).Block(
					Line().Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
					Line().Id("span").Dot("SetTag").Call(Lit("msg"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call()),
					Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
				)
			}
			return Line().If(Err().Op("!=").Nil()).Block(
				Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
			)
		}))
		bg.Add(method.httpCookies(func(arg, header string) *Statement {
			if svc.tags.IsSet(tagTrace) {
				return Line().Id("span").Dot("SetTag").Call(Lit(header), Id("_"+arg)).
					Line().If(Err().Op("!=").Nil()).Block(
					Line().Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
					Line().Id("span").Dot("SetTag").Call(Lit("msg"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call()),
					Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
				)
			}
			return Line().If(Err().Op("!=").Nil()).Block(
				Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
			)
		}))
		bg.Var().Id("response").Id(method.responseStructName())
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
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Err())
				ig.Id("span").Dot("SetTag").Call(Lit("errData"), Id("toString").Call(Err()))
			}
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

		bg.If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Marshal").Call(Id("response")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call())
			}
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
		})
		bg.Return()
	})
}

func (svc *service) serveMethodFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serveMethod").
		ParamsFunc(func(pg *Group) {
			pg.Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")
			pg.Id("methodName").String()
			pg.Id("methodHandler").Id("methodJsonRPC")
		}).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			if svc.tags.IsSet(tagTrace) {
				bg.Id("span").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id(_ctx_).Dot("UserContext").Call())
				bg.Id("span").Dot("SetTag").Call(Lit("method"), Id("methodName")).Line()
			}
			bg.Id("methodHTTP").Op(":=").Id(_ctx_).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(packageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("only POST method supported"))
				}
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
			})
			bg.Var().Id("request").Id("baseJsonRPC")
			bg.Var().Id("response").Op("*").Id("baseJsonRPC")
			bg.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			})
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))

			bg.If(Id("method").Op("!=").Lit("").Op("&&").Id("method").Op("!=").Id("methodName")).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method ").Op("+").Id("methodNameOrigin"))
				}
				ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method ").Op("+").Id("methodNameOrigin"), Nil()))
			})
			bg.Id("response").Op("=").Id("methodHandler").Call(Id(_ctx_), Id("request"))
			bg.If(Id("response").Op("!=").Nil()).Block(
				Return().Id("sendResponse").Call(Id(_ctx_), Id("response")),
			)
			bg.Return()
		})
}

func (svc *service) batchFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id("doBatch").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requests").Op("[]").Id("baseJsonRPC")).Params(Id("responses").Id("jsonrpcResponses")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.If(Len(Id("requests")).Op(">").Id("http").Dot("maxBatchSize")).Block(
				Id("responses").Dot("append").Call(Id("makeErrorResponseJsonRPC").Call(Nil(), Id("invalidRequestError"), Lit("batch size exceeded"), Nil())),
				Return(),
			)
			bg.If(Qual(packageStrings, "EqualFold").Call(Id(_ctx_).Dot("Get").Call(Lit(syncHeader)), Lit("true"))).Block(
				For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
					Id("response").Op(":=").Id("http").Dot("doSingleBatch").Call(Id(_ctx_), Id("request")),
					If(Id("request").Dot("ID").Op("!=").Nil()).Block(
						Id("responses").Dot("append").Call(Id("response")),
					),
				),
				Return(),
			)
			bg.Var().Id("wg").Qual(packageSync, "WaitGroup")
			bg.Id("batchSize").Op(":=").Id("http").Dot("maxParallelBatch")
			bg.If(Len(Id("requests")).Op("<").Id("batchSize")).Block(
				Id("batchSize").Op("=").Len(Id("requests")),
			)
			bg.Id("callCh").Op(":=").Make(Chan().Id("baseJsonRPC"), Id("batchSize"))
			bg.Id("responses").Op("=").Make(Id("jsonrpcResponses"), Lit(0), Len(Id("requests")))
			bg.For(Id("i").Op(":=").Lit(0).Op(";").Id("i").Op("<").Id("batchSize").Op(";").Id("i").Op("++")).Block(
				Id("wg").Dot("Add").Call(Lit(1)),
				Go().Func().Params().Block(
					Defer().Id("wg").Dot("Done").Call(),
					For(Id("request").Op(":=").Range().Id("callCh").Block(
						Id("response").Op(":=").Id("http").Dot("doSingleBatch").Call(Id(_ctx_), Id("request")),
						If(Id("request").Dot("ID").Op("!=").Nil()).Block(
							Id("responses").Dot("append").Call(Id("response")),
						),
					)),
				).Call(),
			)
			bg.For(Id("idx").Op(":=").Range().Id("requests").Block(
				Id("callCh").Op("<-").Id("requests").Op("[").Id("idx").Op("]"),
			))
			bg.Close(Id("callCh"))
			bg.Id("wg").Dot("Wait").Call()
			bg.Return()
		})
}

func (svc *service) singleBatchFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id("doSingleBatch").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("request").Id("baseJsonRPC")).Params(Id("response").Op("*").Id("baseJsonRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			if svc.tr.hasTrace() {
				bg.Id("methodContext").Op(":=").Id(_ctx_).Dot("UserContext").Call()
			}
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))
			if svc.tr.hasTrace() {
				bg.Id("batchSpan").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id("methodContext"))
			}
			if svc.tr.hasTrace() {
				bg.Id("span").Op(":=").Qual(packageOpentracing, "StartSpan").
					Call(Id("request").Dot("Method"), Qual(packageOpentracing, "ChildOf").Call(Id("batchSpan").Dot("Context").Call()))
				bg.Defer().Id("span").Dot("Finish").Call()
				bg.Id("methodContext").Op("=").Qual(packageOpentracing, "ContextWithSpan").Call(Id(_ctx_).Dot("UserContext").Call(), Id("span"))
			}
			bg.Switch(Id("method")).BlockFunc(
				func(sg *Group) {
					for _, method := range svc.methods {
						if !method.isJsonRPC() {
							continue
						}
						sg.Case(Lit(method.lcName())).Block(
							Return(Id("http").Dot(utils.ToLowerCamel(method.Name)).Call(Id(_ctx_), Id("request"))),
						)
					}
					sg.Default().BlockFunc(func(dg *Group) {
						if svc.tr.hasTrace() {
							dg.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
							dg.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"))
						}
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
			if svc.tr.hasTrace() {
				bg.Id("batchSpan").Op(":=").Qual(packageOpentracing, "SpanFromContext").Call(Id(_ctx_).Dot("UserContext").Call())
			}
			bg.Id("methodHTTP").Op(":=").Id(_ctx_).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(packageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				if svc.tr.hasTrace() {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True())
					ig.Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("only POST method supported"))
				}
				ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				ig.Return()
			})
			bg.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Var().Id("request").Id("baseJsonRPC")
				ig.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					if svc.tr.hasTrace() {
						ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True())
						ig.Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
					}
					ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
				})
				ig.Id("single").Op("=").True()
				ig.Id("requests").Op("=").Append(Id("requests"), Id("request"))
			})
			bg.If(Id("single")).Block(
				Return(Id("sendResponse").Call(Id(_ctx_), Id("http").Dot("doSingleBatch").
					Call(Id(_ctx_), Id("requests").Op("[").Lit(0).Op("]")),
				)),
			)
			bg.Return(Id("sendResponse").Call(Id(_ctx_), Id("http").Dot("doBatch").
				Call(Id(_ctx_), Id("requests")),
			))
		})
}
