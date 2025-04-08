// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-jsonrpc.go at 25.06.2020, 11:07) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (tr *Transport) renderJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageErrors, "errors")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(tr.tags.Value(tagPackageJSON, packageStdJSON), "json")

	srcFile.Line().Add(tr.jsonrpcConstants(false))
	srcFile.Add(tr.idJsonRPC()).Line()
	srcFile.Add(tr.baseJsonRPC(false)).Line()
	srcFile.Add(tr.errorJsonRPC()).Line()
	srcFile.Add(tr.jsonrpcResponsesTypeFunc())

	srcFile.Add(tr.serveBatchFunc())
	srcFile.Add(tr.batchFunc())
	srcFile.Add(tr.singleBatchFunc())

	srcFile.Line().Type().Id("methodJsonRPC").Func().Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).Params(Id("responseBase").Op("*").Id("baseJsonRPC"))
	srcFile.Line().Add(tr.makeErrorResponseJsonRPCFunc())
	return srcFile.Save(path.Join(outDir, "jsonrpc.go"))
}

func (tr *Transport) makeErrorResponseJsonRPCFunc() Code {

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

func (tr *Transport) baseJsonRPC(isClient bool) Code {

	return Type().Id("baseJsonRPC").StructFunc(func(tg *Group) {

		tg.Id("ID").Id("idJsonRPC").Tag(map[string]string{"json": "id"})
		tg.Id("Version").Id("string").Tag(map[string]string{"json": "jsonrpc"})
		tg.Id("Method").Id("string").Tag(map[string]string{"json": "method,omitempty"})

		if isClient {
			tg.Id("Error").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "RawMessage").Tag(map[string]string{"json": "error,omitempty"})
			tg.Id("Params").Interface().Tag(map[string]string{"json": "params,omitempty"})
		} else {
			tg.Id("Error").Op("*").Id("errorJsonRPC").Tag(map[string]string{"json": "error,omitempty"})
			tg.Id("Params").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "RawMessage").Tag(map[string]string{"json": "params,omitempty"})
		}

		tg.Id("Result").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "RawMessage").Tag(map[string]string{"json": "result,omitempty"})

		if isClient {
			tg.Line().Id("retHandler").Func().Params(Id("baseJsonRPC"))
		}
	})
}

func (tr *Transport) errorJsonRPC() Code {

	return Type().Id("errorJsonRPC").Struct(
		Id("Code").Id("int").Tag(map[string]string{"json": "code"}),
		Id("Message").Id("string").Tag(map[string]string{"json": "message"}),
		Id("Data").Id("interface{}").Tag(map[string]string{"json": "data,omitempty"}),
	).Line().Func().Params(Err().Id("errorJsonRPC")).Id("Error").Params().Params(String()).Block(
		Return(Err().Dot("Message")),
	)
}

func (tr *Transport) jsonrpcResponsesTypeFunc() Code {

	return Type().Id("jsonrpcResponses").Op("[]").Op("*").Id("baseJsonRPC").
		Line().Func().Params(Id("responses").Op("*").Id("jsonrpcResponses")).Id("append").Params(Id("response").Op("*").Id("baseJsonRPC")).Block(
		If(Id("response").Op("==").Nil()).Block(Return()),
		If(Id("response").Dot("ID").Op("!=").Nil()).Block(
			Op("*").Id("responses").Op("=").Append(Op("*").Id("responses"), Id("response")),
		),
	)
}

func (tr *Transport) idJsonRPC() Code {
	return Type().Id("idJsonRPC").Op("=").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "RawMessage")
}

func (tr *Transport) jsonrpcConstants(exportErrors bool) Code {

	export := func(name string, export bool) string {
		if export {
			name = utils.ToCamel(name)
		}
		return name
	}
	return Const().Op("(").
		Line().Id("defaultMaxBatchSize").Op("=").Lit(100).
		Line().Id("defaultMaxParallelBatch").Op("=").Lit(10).
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

func (tr *Transport) singleBatchFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("doSingleBatch").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("request").Id("baseJsonRPC")).Params(Id("response").Op("*").Id("baseJsonRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Qual(packageStrings, "ToLower").Call(Id("request").Dot("Method"))
			bg.Switch(Id("method")).BlockFunc(
				func(sg *Group) {
					for _, serviceName := range tr.serviceKeys() {
						svc := tr.services[serviceName]
						for _, method := range svc.methods {
							if !method.isJsonRPC() {
								continue
							}
							sg.Case(Lit(svc.lcName() + "." + method.lcName())).Block(
								Return(Id("srv").Dot("http"+serviceName).Dot(utils.ToLowerCamel(method.Name)).Call(Id(_ctx_), Id("request"))),
							)
						}
					}
					sg.Default().BlockFunc(func(dg *Group) {
						dg.Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil()))
					})
				})
		})
}

func (tr *Transport) batchFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("doBatch").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requests").Op("[]").Id("baseJsonRPC")).Params(Id("responses").Id("jsonrpcResponses")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.If(Len(Id("requests")).Op(">").Id("srv").Dot("maxBatchSize")).Block(
				Id("responses").Dot("append").Call(Id("makeErrorResponseJsonRPC").Call(Nil(), Id("invalidRequestError"), Lit("batch size exceeded"), Nil())),
				Return(),
			)
			bg.If(Qual(packageStrings, "EqualFold").Call(Id(_ctx_).Dot("Get").Call(Lit(syncHeader)), Lit("true"))).Block(
				For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
					Id("response").Op(":=").Id("srv").Dot("doSingleBatch").Call(Id(_ctx_), Id("request")),
					If(Id("request").Dot("ID").Op("!=").Nil()).Block(
						Id("responses").Dot("append").Call(Id("response")),
					),
				),
				Return(),
			)
			bg.Var().Id("wg").Qual(packageSync, "WaitGroup")
			bg.Id("batchSize").Op(":=").Id("srv").Dot("maxParallelBatch")
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
						Id("response").Op(":=").Id("srv").Dot("doSingleBatch").Call(Id(_ctx_), Id("request")),
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

func (tr *Transport) serveBatchFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).Id("serveBatch").
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
			bg.If(Err().Op("=").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Var().Id("request").Id("baseJsonRPC")
				ig.If(Err().Op("=").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id(_ctx_).Dot("Body").Call(), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.Return().Id("sendResponse").Call(Id(_ctx_), Id("makeErrorResponseJsonRPC").Call(Op("[]").Byte().Call(Lit(`"0"`)), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
				})
				ig.Id("single").Op("=").True()
				ig.Id("requests").Op("=").Append(Id("requests"), Id("request"))
			})
			bg.If(Id("single")).Block(
				Return(Id("sendResponse").Call(Id(_ctx_), Id("srv").Dot("doSingleBatch").
					Call(Id(_ctx_), Id("requests").Op("[").Lit(0).Op("]")),
				)),
			)
			bg.Return(Id("sendResponse").Call(Id(_ctx_), Id("srv").Dot("doBatch").
				Call(Id(_ctx_), Id("requests")),
			))
		})
}
