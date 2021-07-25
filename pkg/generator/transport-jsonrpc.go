// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-jsonrpc.go at 25.06.2020, 11:07) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/pkg/utils"
)

func (tr Transport) renderJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageOpentracingExt, "ext")
	srcFile.ImportName(packageOpentracing, "opentracing")

	srcFile.Line().Add(tr.jsonrpcConstants(false))
	srcFile.Add(tr.idJsonRPC()).Line()
	srcFile.Add(tr.baseJsonRPC(false)).Line()
	srcFile.Add(tr.errorJsonRPC()).Line()
	srcFile.Add(tr.jsonrpcResponsesTypeFunc())

	hasTrace := tr.hasTrace()
	srcFile.Line().Type().Id("methodJsonRPC").Func().Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).Params(Id("responseBase").Op("*").Id("baseJsonRPC"))
	if hasTrace {
		srcFile.Type().Id("methodTraceJsonRPC").Func().Params(Id("span").Qual(packageOpentracing, "Span"), Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).Params(Id("responseBase").Op("*").Id("baseJsonRPC"))
	}
	srcFile.Add(tr.serveBatchFunc(hasTrace))
	srcFile.Line().Add(tr.makeErrorResponseJsonRPCFunc())
	return srcFile.Save(path.Join(outDir, "jsonrpc.go"))
}

func (tr Transport) serveBatchFunc(hasTrace bool) Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("serveBatch").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Err().Error()).BlockFunc(func(bg *Group) {

		if hasTrace {
			bg.Id("batchSpan").Op(":=").Id("extractSpan").Call(Id("srv").Dot("log"), Qual(packageFmt, "Sprintf").Call(Lit("jsonRPC:%s"), Id(_ctx_).Dot("Path").Call()), Id(_ctx_))
			bg.Defer().Id("injectSpan").Call(Id("srv").Dot("log"), Id("batchSpan"), Id(_ctx_))
			bg.Defer().Id("batchSpan").Dot("Finish").Call()
		}

		bg.Id("methodHTTP").Op(":=").Id(_ctx_).Dot("Method").Call()

		bg.If(Id("methodHTTP").Op("!=").Qual(packageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
			if hasTrace {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True())
				ig.Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("only POST method supported"))
			}
			ig.Id(_ctx_).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(packageFiber, "StatusMethodNotAllowed"))
			ig.If(List(Id("_"), Err()).Op("=").Id(_ctx_).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			)
			ig.Return()
		})
		bg.If(Id("value").Op(":=").Id(_ctx_).Dot("Context").Call().Dot("Value").Call(Id("CtxCancelRequest")).Op(";").Id("value").Op("!=").Nil()).Block(
			Return(),
		)
		bg.Id(_ctx_).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Id("contentTypeJson"))
		bg.Var().Id("requests").Op("[]").Id("baseJsonRPC")
		bg.If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			if hasTrace {
				ig.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("batchSpan"), True())
				ig.Id("batchSpan").Dot("SetTag").Call(Lit("msg"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
			}
			ig.Id("sendResponse").Call(Id("srv").Dot("log"), Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			ig.Return()
		})
		bg.Id("responses").Op(":=").Make(Id("jsonrpcResponses"), Lit(0), Len(Id("requests")))
		bg.Var().Id("n").Int()
		bg.Var().Id("wg").Qual(packageSync, "WaitGroup")
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
			Id("methodNameOrigin").Op(":=").Id("request").Dot("Method"),
			Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method")),
			Switch(Id("method")).BlockFunc(func(bg *Group) {
				for serviceName, svc := range tr.services {
					for _, method := range svc.methods {
						if !method.isJsonRPC() {
							continue
						}
						bg.Line().Case(Lit(svc.lcName()+"."+method.lcName())).Block(
							Id("wg").Dot("Add").Call(Lit(1)),
							Go().Func().Params(Id("request").Id("baseJsonRPC")).BlockFunc(func(gg *Group) {
								if svc.tags.IsSet(tagTrace) {
									gg.Line().Id("span").Op(":=").Qual(packageOpentracing, "StartSpan").Call(Id("request").Dot("Method"), Qual(packageOpentracing, "ChildOf").Call(Id("batchSpan").Dot("Context").Call()))
									gg.Id("span").Dot("SetTag").Call(Lit("batch"), True())
									gg.Defer().Id("span").Dot("Finish").Call()
								}
								gg.If(Id("request").Dot("ID").Op("!=").Nil()).BlockFunc(func(ig *Group) {
									if svc.tags.IsSet(tagTrace) {
										ig.Id("responses").Dot("append").Call(Id("srv").Dot("http"+serviceName).Dot(utils.ToLowerCamel(method.Name)).Call(Id("span"), Id(_ctx_), Id("request")))
									} else {
										ig.Id("responses").Dot("append").Call(Id("srv").Dot("http"+serviceName).Dot(utils.ToLowerCamel(method.Name)).Call(Id(_ctx_), Id("request")))
									}
									ig.Id("wg").Dot("Done").Call()
									ig.Return()
								})
								if svc.tags.IsSet(tagTrace) {
									gg.Id("srv").Dot("http"+serviceName).Dot(utils.ToLowerCamel(method.Name)).Call(Id("span"), Id(_ctx_), Id("request"))
								} else {
									gg.Id("srv").Dot("http"+serviceName).Dot(utils.ToLowerCamel(method.Name)).Call(Id(_ctx_), Id("request"))
								}
								gg.Id("wg").Dot("Done").Call()
							}).Call(Id("request")),
						)
					}
				}
				bg.Default().BlockFunc(func(dg *Group) {
					if hasTrace {
						dg.Id("span").Op(":=").Qual(packageOpentracing, "StartSpan").Call(Id("request").Dot("Method"), Qual(packageOpentracing, "ChildOf").Call(Id("batchSpan").Dot("Context").Call()))
						dg.Id("span").Dot("SetTag").Call(Lit("batch"), True())
						dg.Qual(packageOpentracingExt, "Error").Dot("Set").Call(Id("span"), True())
						dg.Id("span").Dot("SetTag").Call(Lit("msg"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"))
					}
					dg.Id("responses").Dot("append").Call(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil()))
					if hasTrace {
						dg.Id("span").Dot("Finish").Call()
					}
				})
			}),
			If(Id("n").Op(">").Id("maxParallelBatch")).Block(
				Id("n").Op("=").Lit(0),
				Id("wg").Dot("Wait").Call(),
			),
			Id("n").Op("++"),
		)
		bg.Id("wg").Dot("Wait").Call()
		bg.Id("sendResponse").Call(Id("srv").Dot("log"), Id(_ctx_), Id("responses"))
		bg.Return()
	})
}

func (tr Transport) makeErrorResponseJsonRPCFunc() Code {

	return Func().Id("makeErrorResponseJsonRPC").Params(Id("id").Id("idJsonRPC"), Id("code").Int(), Id("msg").String(), Id("data").Interface()).Params(Op("*").Id("baseJsonRPC")).Block(

		Line().If(Id("id").Op("==").Nil()).Block(
			Return(Nil()),
		),

		Line().Return(Op("&").Id("baseJsonRPC").Values(Dict{
			Id("ID"):      Id("id"),
			Id("Version"): Id("Version"),
			Id("Error"): Op("&").Id("errorJsonRPC").Values(Dict{
				Id("Code"):    Id("code"),
				Id("Message"): Id("msg"),
				Id("Data"):    Id("data"),
			}),
		})),
	)
}

func (tr Transport) baseJsonRPC(isClient bool) Code {

	return Type().Id("baseJsonRPC").StructFunc(func(tg *Group) {

		tg.Id("ID").Id("idJsonRPC").Tag(map[string]string{"json": "id"})
		tg.Id("Version").Id("string").Tag(map[string]string{"json": "jsonrpc"})
		tg.Id("Method").Id("string").Tag(map[string]string{"json": "method,omitempty"})

		if isClient {
			tg.Id("Error").Qual(packageJson, "RawMessage").Tag(map[string]string{"json": "error,omitempty"})
			tg.Id("Params").Interface().Tag(map[string]string{"json": "params,omitempty"})
		} else {
			tg.Id("Error").Op("*").Id("errorJsonRPC").Tag(map[string]string{"json": "error,omitempty"})
			tg.Id("Params").Qual(packageJson, "RawMessage").Tag(map[string]string{"json": "params,omitempty"})
		}

		tg.Id("Result").Qual(packageJson, "RawMessage").Tag(map[string]string{"json": "result,omitempty"})

		if isClient {
			tg.Line().Id("retHandler").Func().Params(Id("baseJsonRPC"))
		}
	})
}

func (tr Transport) errorJsonRPC() Code {

	return Type().Id("errorJsonRPC").Struct(
		Id("Code").Id("int").Tag(map[string]string{"json": "code"}),
		Id("Message").Id("string").Tag(map[string]string{"json": "message"}),
		Id("Data").Id("interface{}").Tag(map[string]string{"json": "data,omitempty"}),
	).Line().Func().Params(Err().Id("errorJsonRPC")).Id("Error").Params().Params(String()).Block(
		Return(Err().Dot("Message")),
	)
}

func (tr Transport) jsonrpcResponsesTypeFunc() Code {

	return Type().Id("jsonrpcResponses").Op("[]").Id("baseJsonRPC").
		Line().Func().Params(Id("responses").Op("*").Id("jsonrpcResponses")).Id("append").Params(Id("response").Op("*").Id("baseJsonRPC")).Block(
		If(Id("response").Op("==").Nil()).Block(Return()),
		If(Id("response").Dot("ID").Op("!=").Nil()).Block(
			Op("*").Id("responses").Op("=").Append(Op("*").Id("responses"), Op("*").Id("response")),
		),
	)
}

func (tr Transport) idJsonRPC() Code {
	return Type().Id("idJsonRPC").Op("=").Qual(packageJson, "RawMessage")
}

func (tr Transport) jsonrpcConstants(exportErrors bool) Code {

	export := func(name string, export bool) string {

		if export {
			name = utils.ToCamel(name)
		}
		return name
	}

	return Const().Op("(").
		Line().Id("maxParallelBatch").Op("=").Lit(100).
		Line().Comment("Version defines the version of the JSON RPC implementation").
		Line().Id("Version").Op("=").Lit("2.0").
		Line().Comment("contentTypeJson defines the content type to be served").
		Line().Id("contentTypeJson").Op("=").Lit("application/json").
		Line().Comment("ParseError defines invalid JSON was received by the server").
		Line().Comment("An error occurred on the server while parsing the JSON text").
		Line().Id(export("parseError", exportErrors)).Op("=").Lit(-32700).
		Line().Comment("InvalidRequestError defines the JSON sent is not a valid Request object").
		Line().Id(export("invalidRequestError", exportErrors)).Op("=").Lit(-32600).
		Line().Comment("MethodNotFoundError defines the method does not exist / is not available").
		Line().Id(export("methodNotFoundError", exportErrors)).Op("=").Lit(-32601).
		Line().Comment("InvalidParamsError defines invalid method parameter(s)").
		Line().Id(export("invalidParamsError", exportErrors)).Op("=").Lit(-32602).
		Line().Comment("InternalError defines a server error").
		Line().Id(export("internalError", exportErrors)).Op("=").Lit(-32603).
		Op(")")
}
