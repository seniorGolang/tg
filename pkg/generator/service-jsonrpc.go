// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (service-jsonrpc.go at 24.06.2020, 15:31) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
	"github.com/vetcher/go-astra/types"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	srcFile.ImportName(packageOpentracing, "opentracing")

	for _, method := range svc.methods {
		if !method.isJsonRPC() {
			continue
		}
		srcFile.Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serve" + method.Name).Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Err().Error()).Block(
			Return().Id("http").Dot("serveMethod").Call(Id(_ctx_), Lit(method.lcName()), Id("http").Dot(method.lccName())),
		)
		srcFile.Add(svc.rpcMethodFunc(method))
	}
	srcFile.Add(svc.serveServiceBatchFunc())
	srcFile.Add(svc.serveMethodFunc())
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-jsonrpc.go"))
}

func (svc *service) serveServiceBatchFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serveBatch").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Err().Error()).BlockFunc(func(bg *Group) {

		if svc.tags.IsSet(tagTrace) {
			bg.Id("batchSpan").Op(":=").Id("extractSpan").Call(Id("http").Dot("log"), Qual(packageFmt, "Sprintf").Call(Lit("jsonRPC:%s"), Id(_ctx_).Dot("Path").Call()), Id(_ctx_))
			bg.Defer().Id("injectSpan").Call(Id("http").Dot("log"), Id("batchSpan"), Id(_ctx_))
			bg.Defer().Id("batchSpan").Dot("Finish").Call()
		}
		bg.Id("methodHTTP").Op(":=").Id(_ctx_).Dot("Method").Call()
		bg.If(Id("methodHTTP").Op("!=").Qual(packageFiber, "MethodPost")).BlockFunc(func(mg *Group) {
			if svc.tags.IsSet(tagTrace) {
				mg.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True())
				mg.Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("only POST method supported"))
			}
			mg.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusMethodNotAllowed"))
			mg.If(List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			)
			mg.Return()
		})
		bg.If(Id("value").Op(":=").Id(_ctx_).Dot("Context").Call().Dot("Value").Call(Id("CtxCancelRequest")).Op(";").Id("value").Op("!=").Nil()).Block(
			Return(),
		)
		bg.Var().Id("single").Bool()
		bg.Var().Id("requests").Op("[]").Id("baseJsonRPC")
		bg.If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			ig.Var().Id("request").Id("baseJsonRPC")
			ig.If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True())
					ig.Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			})
			ig.Id("single").Op("=").True()
			ig.Id("requests").Op("=").Append(Id("requests"), Id("request"))
		})

		bg.Id("responses").Op(":=").Make(Id("jsonrpcResponses"), Lit(0), Len(Id("requests")))
		bg.Var().Id("wg").Qual(packageSync, "WaitGroup")
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).BlockFunc(func(fg *Group) {
			fg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			fg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))
			if svc.tags.IsSet(tagTrace) {
				fg.Id("span").Op(":=").Qual(packageOpentracing, "StartSpan").Call(Id("request").Dot("Method"), Qual(packageOpentracing, "ChildOf").Call(Id("batchSpan").Dot("Context").Call()))
				fg.Id("span").Dot("SetTag").Call(Lit("batch"), True())
			}
			fg.Switch(Id("method")).BlockFunc(func(bg *Group) {
				for _, method := range svc.methods {
					if !method.isJsonRPC() {
						continue
					}
					bg.Case(Lit(method.lcName())).Block(
						Id("wg").Dot("Add").Call(Lit(1)),
						Func().Params(Id("request").Id("baseJsonRPC")).BlockFunc(func(gr *Group) {
							if svc.tags.IsSet(tagTrace) {
								gr.Id("responses").Dot("append").Call(Id("http").Dot(method.lccName()).Call(Id("span"), Id(_ctx_), Id("request")))
							} else {
								gr.Id("responses").Dot("append").Call(Id("http").Dot(method.lccName()).Call(Id(_ctx_), Id("request")))
							}
							gr.Id("wg").Dot("Done").Call()
						}).Call(Id("request")),
					)
				}
				bg.Default().BlockFunc(func(bf *Group) {
					if svc.tags.IsSet(tagTrace) {
						bf.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
						bf.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"))
					}
					bf.Id("responses").Dot("append").Call(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil()))
				})
			})
			if svc.tags.IsSet(tagTrace) {
				fg.Id("span").Dot("Finish").Call()
			}
		})
		bg.Id("wg").Dot("Wait").Call()
		bg.If(Id("single")).Block(
			Return().Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("responses").Op("[").Lit(0).Op("]")),
		)
		bg.Return().Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("responses"))
	})
}

func (svc *service) rpcMethodFunc(method *method) Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id(method.lccName()).
		ParamsFunc(func(pg *Group) {
			if svc.tags.IsSet(tagTrace) {
				pg.Id("span").Qual(packageOpentracing, "Span")
			}
			pg.Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")
			pg.Id("requestBase").Id("baseJsonRPC")
		}).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC")).BlockFunc(func(bf *Group) {

		bf.Var().Err().Error()
		bf.Var().Id("request").Id(method.requestStructName())

		bf.If(Id("requestBase").Dot("Params").Op("!=").Nil()).Block(
			If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			}),
		)
		bf.If(Id("requestBase").Dot("Version").Op("!=").Id("Version")).BlockFunc(func(ig *Group) {
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version"))
			}
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version"), Nil()))
		})
		if svc.tags.IsSet(tagTrace) {
			bf.Id("methodContext").Op(":=").Qual(packageOpentracing, "ContextWithSpan").Call(Id(_ctx_).Dot("Context").Call(), Id("span"))
		} else {
			bf.Id("methodContext").Op(":=").Id(_ctx_).Dot("UserContext").Call()
		}
		bf.Add(method.httpArgHeaders(func(arg, header string) *Statement {
			if svc.tags.IsSet(tagTrace) {
				return Line().Id("methodContext").Op("=").Qual(packageContext, "WithValue").Call(Id("methodContext"), Lit(header), Id("_"+arg)).
					Line().Id("span").Dot("SetTag").Call(Lit(header), Id("_"+arg)).
					Line().If(Err().Op("!=").Nil()).Block(
					Line().Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
					Line().Id("span").Dot("SetTag").Call(Lit("msg"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call()),
					Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
				)
			}
			return Line().Id("methodContext").Op("=").Qual(packageContext, "WithValue").Call(Id("methodContext"), Lit(header), Id("_"+arg)).
				Line().If(Err().Op("!=").Nil()).Block(
				Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
			)
		}))
		bf.Add(method.httpCookies(func(arg, header string) *Statement {
			if svc.tags.IsSet(tagTrace) {
				return Line().Id("methodContext").Op("=").Qual(packageContext, "WithValue").Call(Id("methodContext"), Lit(header), Id("_"+arg)).
					Line().Id("span").Dot("SetTag").Call(Lit(header), Id("_"+arg)).
					Line().If(Err().Op("!=").Nil()).Block(
					Line().Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
					Line().Id("span").Dot("SetTag").Call(Lit("msg"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call()),
					Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
				)
			}
			return Line().Id("methodContext").Op("=").Qual(packageContext, "WithValue").Call(Id("methodContext"), Lit(header), Id("_"+arg)).
				Line().If(Err().Op("!=").Nil()).Block(
				Line().Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
			)
		}))
		bf.Var().Id("response").Id(method.responseStructName())
		bf.ListFunc(func(lg *Group) {
			for _, ret := range method.resultsWithoutError() {
				lg.Id("response").Dot(utils.ToCamel(ret.Name))
			}
			lg.Err()
		}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
			cg.Id("methodContext")
			for _, arg := range method.argsWithoutContext() {

				argCode := Id("request").Dot(utils.ToCamel(arg.Name))

				if types.IsEllipsis(arg.Type) {
					argCode.Op("...")
				}
				cg.Add(argCode)
			}
		})
		bf.If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			ig.If(Id("http").Dot("errorHandler").Op("!=").Nil()).Block(
				Err().Op("=").Id("http").Dot("errorHandler").Call(Err()),
			)
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Err())
				ig.Id("span").Dot("SetTag").Call(Lit("errData"), Id("toString").Call(Err()))
			}
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("internalError"), Err().Dot("Error").Call(), Err()))
		})
		bf.Id("responseBase").Op("=").Op("&").Id("baseJsonRPC").Values(Dict{
			Id("Version"): Id("Version"),
			Id("ID"):      Id("requestBase").Dot("ID"),
		})

		bf.If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(packageJson, "Marshal").Call(Id("response")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			if svc.tags.IsSet(tagTrace) {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
				ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call())
			}
			ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
		})
		bf.Return()
	})
}

func (svc *service) serveMethodFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serveMethod").
		ParamsFunc(func(pg *Group) {
			pg.Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")
			pg.Id("methodName").String()
			if svc.tags.IsSet(tagTrace) {
				pg.Id("methodHandler").Id("methodTraceJsonRPC")
			} else {
				pg.Id("methodHandler").Id("methodJsonRPC")
			}
		}).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			if svc.tags.IsSet(tagTrace) {
				bg.Id("span").Op(":=").Id("extractSpan").Call(
					Id("http").Dot("log"),
					Qual(packageFmt, "Sprintf").Call(Lit("jsonRPC:%s"), Id(_ctx_).Dot("Path").Call()),
					Id(_ctx_),
				)
				bg.Defer().Id("injectSpan").Call(Id("http").Dot("log"), Id("span"), Id(_ctx_))
				bg.Defer().Id("span").Dot("Finish").Call()
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
			bg.If(Id("value").Op(":=").Id(_ctx_).Dot("Context").Call().Dot("Value").Call(Id("CtxCancelRequest")).Op(";").Id("value").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request canceled"))
				}
				ig.Return()
			})
			bg.Var().Id("request").Id("baseJsonRPC")
			bg.Var().Id("response").Op("*").Id("baseJsonRPC")
			bg.If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
				}
				ig.Return().Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			})
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))

			bg.If(Id("method").Op("!=").Lit("").Op("&&").Id("method").Op("!=").Id("methodName")).BlockFunc(func(ig *Group) {
				if svc.tags.IsSet(tagTrace) {
					ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
					ig.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method ").Op("+").Id("methodNameOrigin"))
				}
				ig.Return().Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method ").Op("+").Id("methodNameOrigin"), Nil()))
			})
			if svc.tags.IsSet(tagTrace) {
				bg.Id("response").Op("=").Id("methodHandler").Call(Id("span"), Id(_ctx_), Id("request"))
			} else {
				bg.Id("response").Op("=").Id("methodHandler").Call(Id(_ctx_), Id("request"))
			}
			bg.If(Id("response").Op("!=").Nil()).Block(
				Return().Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("response")),
			)
			bg.Return()
		})
}
