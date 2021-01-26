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

	"github.com/seniorGolang/tg/pkg/utils"
)

func (svc *service) renderJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageGotils, "gotils")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageFastHttp, "fasthttp")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportName(svc.pkgPath, filepath.Base(svc.pkgPath))
	srcFile.ImportName(packageOpentracing, "opentracing")

	for _, method := range svc.methods {

		if !method.isJsonRPC() {
			continue
		}

		srcFile.Func().Params(Id("http").Op("*").Id("http" + svc.Name)).Id("serve" + method.Name).Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(
			Id("http").Dot("serveMethod").Call(Id(_ctx_), Lit(method.lcName()), Id("http").Dot(method.lccName())),
		)
		srcFile.Add(svc.rpcMethodFunc(method))
	}

	srcFile.Line().Add(svc.serveServiceBatchFunc())
	srcFile.Line().Add(svc.serveMethodFunc())

	return srcFile.Save(path.Join(outDir, svc.lcName()+"-jsonrpc.go"))
}

func (svc *service) serveServiceBatchFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id("serveBatch").Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx")).Block(

		Line().Id("batchSpan").Op(":=").Id("extractSpan").Call(Id("http").Dot("log"), Qual(packageFmt, "Sprintf").Call(Lit("jsonRPC:%s"), Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("URI").Call().Dot("Path").Call())), Id(_ctx_)),
		Defer().Id("injectSpan").Call(Id("http").Dot("log"), Id("batchSpan"), Id(_ctx_)),
		Defer().Id("batchSpan").Dot("Finish").Call(),

		Id("methodHTTP").Op(":=").Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Method").Call()),

		Line().If(Id("methodHTTP").Op("!=").Qual(packageFastHttp, "MethodPost")).Block(
			Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True()),
			Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("only POST method supported")),
			Id(_ctx_).Dot("Error").Call(Lit("only POST method supported"), Qual(packageFastHttp, "StatusMethodNotAllowed")),
			Return(),
		),

		Line().If(Id("value").Op(":=").Id(_ctx_).Dot("Value").Call(Id("CtxCancelRequest")).Op(";").Id("value").Op("!=").Nil()).Block(
			Return(),
		),

		Line().Var().Err().Error(),
		Var().Id("requests").Op("[]").Id("baseJsonRPC"),

		Line().If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("PostBody").Call(), Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).Block(
			Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True()),
			Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call()),
			Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())),
			Return(),
		),

		Line().Id("responses").Op(":=").Make(Id("jsonrpcResponses"), Lit(0), Len(Id("requests"))),

		Line().Var().Id("wg").Qual(packageSync, "WaitGroup"),

		Line().For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(

			Line().Id("methodNameOrigin").Op(":=").Id("request").Dot("Method"),
			Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method")),

			Line().Id("span").Op(":=").Qual(packageOpentracing, "StartSpan").Call(Id("request").Dot("Method"), Qual(packageOpentracing, "ChildOf").Call(Id("batchSpan").Dot("Context").Call())),
			Id("span").Dot("SetTag").Call(Lit("batch"), True()),

			Line().Switch(Id("method")).BlockFunc(func(bg *Group) {

				for _, method := range svc.methods {

					if !method.isJsonRPC() {
						continue
					}

					bg.Line().Case(Lit(method.lcName())).Block(

						Line().Id("wg").Dot("Add").Call(Lit(1)),

						Func().Params(Id("request").Id("baseJsonRPC")).Block(
							Id("responses").Dot("append").Call(Id("http").Dot(method.lccName()).Call(Id("span"), Id(_ctx_), Id("request"))),
							Id("wg").Dot("Done").Call(),
						).Call(Id("request")),
					)
				}
				bg.Line().Default().Block(
					Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
					Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'")),
					Id("responses").Dot("append").Call(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil())),
				)
			}),
			Id("span").Dot("Finish").Call(),
		),
		Id("wg").Dot("Wait").Call(),
		Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("responses")),
	)
}

func (svc *service) rpcMethodFunc(method *method) Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id(method.lccName()).
		Params(Id("span").Qual(packageOpentracing, "Span"), Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC")).Block(

		Line().Var().Err().Error(),
		Var().Id("request").Id(method.requestStructName()),

		Line().If(Id("requestBase").Dot("Params").Op("!=").Nil()).Block(
			If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call()),
				Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())),
			),
		),

		Line().If(Id("requestBase").Dot("Version").Op("!=").Id("Version")).Block(
			Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
			Id("span").Dot("SetTag").Call(Lit("msg"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version")),
			Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("incorrect protocol version: ").Op("+").Id("requestBase").Dot("Version"), Nil())),
		),

		Line().Id("methodContext").Op(":=").Qual(packageOpentracing, "ContextWithSpan").Call(Id(_ctx_), Id("span")),

		method.httpArgHeaders(func(arg, header string) *Statement {

			return Line().Id("methodContext").Op("=").Qual(packageContext, "WithValue").Call(Id("methodContext"), Lit(header), Id("_"+arg)).Line().
				Id("span").Dot("SetTag").Call(Lit(header), Id("_"+arg)).Line().
				Line().If(Err().Op("!=").Nil()).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call()),
				Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
			)
		}),

		method.httpCookies(func(arg, header string) *Statement {

			return Line().Id("methodContext").Op("=").Qual(packageContext, "WithValue").Call(Id("methodContext"), Lit(header), Id("_"+arg)).Line().
				Id("span").Dot("SetTag").Call(Lit(header), Id("_"+arg)).Line().
				Line().If(Err().Op("!=").Nil()).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call()),
				Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit(fmt.Sprintf("http header '%s' could not be decoded: ", header)).Op("+").Err().Dot("Error").Call(), Nil())),
			)
		}),

		Line().Var().Id("response").Id(method.responseStructName()),

		Line().ListFunc(func(lg *Group) {

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
		}),

		Line().If(Err().Op("!=").Nil()).Block(
			If(Id("http").Dot("errorHandler").Op("!=").Nil()).Block(
				Err().Op("=").Id("http").Dot("errorHandler").Call(Err()),
			),
			Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
			Id("span").Dot("SetTag").Call(Lit("msg"), Err()),
			Id("span").Dot("SetTag").Call(Lit("errData"), Id("toString").Call(Err())),
			Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("internalError"), Err().Dot("Error").Call(), Err())),
		),

		Line().Id("responseBase").Op("=").Op("&").Id("baseJsonRPC").Values(Dict{
			Id("Version"): Id("Version"),
			Id("ID"):      Id("requestBase").Dot("ID"),
		}),

		Line().If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(packageJson, "Marshal").Call(Id("response")).Op(";").Err().Op("!=").Nil()).Block(
			Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
			Id("span").Dot("SetTag").Call(Lit("msg"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call()),
			Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil())),
		),
		Return(),
	)
}

func (svc *service) serveMethodFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http"+svc.Name)).Id("serveMethod").
		Params(Id(_ctx_).Op("*").Qual(packageFastHttp, "RequestCtx"), Id("methodName").String(), Id("methodHandler").Id("methodJsonRPC")).
		BlockFunc(func(bg *Group) {

			bg.Line().Id("span").Op(":=").Id("extractSpan").Call(
				Id("http").Dot("log"),
				Qual(packageFmt, "Sprintf").Call(Lit("jsonRPC:%s"), Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("URI").Call().Dot("Path").Call())),
				Id(_ctx_),
			)
			bg.Defer().Id("injectSpan").Call(Id("http").Dot("log"), Id("span"), Id(_ctx_))
			bg.Defer().Id("span").Dot("Finish").Call()

			bg.Line().Id("methodHTTP").Op(":=").Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Method").Call())

			bg.Line().If(Id("methodHTTP").Op("!=").Qual(packageFastHttp, "MethodPost")).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit("only POST method supported")),
				Id(_ctx_).Dot("Error").Call(Lit("only POST method supported"), Qual(packageFastHttp, "StatusMethodNotAllowed")),
			)

			bg.Line().If(Id("value").Op(":=").Id(_ctx_).Dot("Value").Call(Id("CtxCancelRequest")).Op(";").Id("value").Op("!=").Nil()).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request canceled")),
				Return(),
			)

			bg.Line().Var().Err().Error()
			bg.Var().Id("request").Id("baseJsonRPC")
			bg.Var().Id("response").Op("*").Id("baseJsonRPC")

			bg.Line().If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("PostBody").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call()),
				Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())),
				Return(),
			)

			bg.Line().Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))

			bg.Line().If(Id("method").Op("!=").Lit("").Op("&&").Id("method").Op("!=").Id("methodName")).Block(
				Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True()),
				Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method ").Op("+").Id("methodNameOrigin")),
				Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method ").Op("+").Id("methodNameOrigin"), Nil())),
				Return(),
			)

			bg.Line().Id("response").Op("=").Id("methodHandler").Call(Id("span"), Id(_ctx_), Id("request"))
			bg.Line().If(Id("response").Op("!=").Nil()).Block(
				Id("sendResponse").Call(Id("http").Dot("log"), Id(_ctx_), Id("response")),
			)
		})
}
