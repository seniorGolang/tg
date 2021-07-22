// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (client-jsonrpc.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderClientJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageHttp, "http")
	srcFile.ImportName(packageJaegerLog, "log")
	srcFile.ImportName(packageLogrus, "logrus")
	srcFile.ImportName(packageFastHttp, "fasthttp")
	srcFile.ImportAlias(packageOpentracing, "otg")

	srcFile.Line().Add(tr.jsonrpcConstants(true))

	srcFile.Line().Add(tr.idJsonRPC())
	srcFile.Type().Id("ErrorDecoder").Func().Params(Id("errData").Qual(packageJson, "RawMessage")).Params(Error())
	srcFile.Line().Add(tr.baseJsonRPC(true))
	srcFile.Line().Add(tr.errorJsonRPC())
	srcFile.Line().Add(tr.jsonrpcClientStructFunc())
	srcFile.Line().Add(tr.jsonrpcBatchTypeFunc())

	srcFile.Line().Func().Id("New").Params(Id("name").String(), Id("log").Qual(packageLogrus, "FieldLogger"), Id("url").String(), Id("opts").Op("...").Id("Option")).Params(Id("cli").Op("*").Id("ClientJsonRPC")).Block(

		Id("cli").Op("=").Op("&").Id("ClientJsonRPC").Values(Dict{
			Id("name"):         Id("name"),
			Id("log"):          Id("log"),
			Id("url"):          Id("url"),
			Id("client"):       Qual(packageFastHttp, "Client").Values(Dict{}),
			Id("errorDecoder"): Id("defaultErrorDecoder"),
		}),

		Line().For(List(Id("_"), Id("opt")).Op(":=").Range().Id("opts")).Block(
			Id("opt").Call(Id("cli")),
		),
		Return(),
	)

	for _, svc := range tr.services {
		if svc.tags.Contains(tagServerJsonRPC) {
			srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id(svc.Name).Params().Params(Op("*").Id("Client" + svc.Name)).Block(
				Return(Op("&").Id("Client" + svc.Name).Values(Dict{
					Id("ClientJsonRPC"): Id("cli"),
				})),
			)
		}
	}

	srcFile.Line().Func().Id("defaultErrorDecoder").Params(Id("errData").Qual(packageJson, "RawMessage")).Params(Err().Error()).Block(

		Line().Var().Id("jsonrpcError").Id("errorJsonRPC"),
		If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id("errData"), Op("&").Id("jsonrpcError")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		),
		Return(Id("jsonrpcError")),
	)

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("Batch").Params(Id(_ctx_).Qual(packageContext, "Context"), Id("requests").Op("...").Id("baseJsonRPC")).Params(Err().Error()).Block(

		Line().Id("span").Op(":=").Id("extractSpan").Call(Id("cli").Dot("log"), Id(_ctx_), Id("cli").Dot("name")),
		Return(Id("cli").Dot("jsonrpcCall").Call(Id(_ctx_), Id("cli").Dot("log"), Id("span"), Id("requests").Op("..."))),
	)

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("BatchFunc").Params(Id(_ctx_).Qual(packageContext, "Context"), Id("batchFunc").Func().Params(Id("requests").Op("*").Id("Batch"))).Params(Err().Error()).Block(

		Line().Var().Id("requests").Id("Batch"),

		Line().Id("batchFunc").Call(Op("&").Id("requests")),
		Id("span").Op(":=").Id("extractSpan").Call(Id("cli").Dot("log"), Id(_ctx_), Id("cli").Dot("name")),

		Line().Return(Id("cli").Dot("jsonrpcCall").Call(Id(_ctx_), Id("cli").Dot("log"), Id("span"), Id("requests").Op("..."))),
	)

	srcFile.Line().Add(tr.jsonrpcClientCallFunc())

	return srcFile.Save(path.Join(outDir, "jsonrpc.go"))
}

func (tr Transport) jsonrpcClientStructFunc() Code {

	return Type().Id("ClientJsonRPC").Struct(
		Id("url").String(),
		Id("name").String(),
		Id("log").Qual(packageLogrus, "FieldLogger"),
		Id("client").Qual(packageFastHttp, "Client"),
		Id("headers").Op("[]").String(),
		Line().Id("errorDecoder").Id("ErrorDecoder"),
	)
}

func (tr Transport) jsonrpcClientCallFunc() Code {

	return Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("jsonrpcCall").
		Params(Id(_ctx_).Qual(packageContext, "Context"), Id("log").Qual(packageLogrus, "FieldLogger"), Id("span").Qual(packageOpentracing, "Span"), Id("requests").Op("...").Id("baseJsonRPC")).Params(Err().Error()).Block(

		Line().Defer().Id("span").Dot("Finish").Call(),

		Line().Id("req").Op(":=").Qual(packageFastHttp, "AcquireRequest").Call(),
		Id("resp").Op(":=").Qual(packageFastHttp, "AcquireResponse").Call(),

		Line().Id("req").Dot("SetRequestURI").Call(Id("cli").Dot("url")),
		Line().Id("req").Dot("Header").Dot("SetMethod").Call(Qual(packageFastHttp, "MethodPost")),

		Line().List(Id("requestID"), Id("_")).Op(":=").Id(_ctx_).Dot("Value").Call(Id("headerRequestID")).Op(".(").String().Op(")"),
		If(Id("requestID").Op("==").Lit("")).Block(
			Id("requestID").Op("=").Qual(packageUUID, "NewV4").Call().Dot("String").Call(),
		),
		Id("req").Dot("Header").Dot("Set").Call(Id("headerRequestID"), Id("requestID")),
		For(List(Id("_"), Id("header")).Op(":=").Range().Id("cli").Dot("headers")).Block(
			If(List(Id("value"), Id("ok")).Op(":=").Id(_ctx_).Dot("Value").Call(Id("header")).Op(".(").String().Op(")")).Op(";").Id("ok").Block(
				Id("req").Dot("Header").Dot("Set").Call(Id("header"), Id("value")),
			),
		),

		Line().If(Err().Op("=").Qual(packageJson, "NewEncoder").Call(Id("req").Dot("BodyWriter").Call()).Dot("Encode").Call(Id("requests")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		),

		Line().Id("injectSpan").Call(Id("log"), Id("span"), Id("req")),
		If(Err().Op("=").Id("cli").Dot("client").Dot("Do").Call(Id("req"), Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		),

		Id("responseMap").Op(":=").Make(Map(String()).Func().Params(Id("baseJsonRPC"))),

		Line().For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(

			If(Id("request").Dot("ID").Op("!=").Nil()).Block(
				Id("responseMap").Op("[").String().Call(Id("request").Dot("ID")).Op("]").Op("=").Id("request").Dot("retHandler"),
			),
		),

		Line().Var().Id("responses").Op("[]").Id("baseJsonRPC"),

		Line().If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id("resp").Dot("Body").Call(), Op("&").Id("responses")).Op(";").Err().Op("!=").Nil()).Block(
			Id("cli").Dot("log").Dot("WithError").Call(Err()).Dot("WithField").Call(Lit("response"), String().Call(Id("resp").Dot("Body").Call())).Dot("Error").Call(Lit("unmarshal response error")),
			Return(),
		),

		Line().For(List(Id("_"), Id("response")).Op(":=").Range().Id("responses")).Block(

			If(List(Id("handler"), Id("found")).Op(":=").Id("responseMap").Op("[").String().Call(Id("response").Dot("ID")).Op("]").Op(";").Id("found")).Block(
				Id("handler").Call(Id("response")),
			),
		),
		Return(),
	)
}

func (tr Transport) jsonrpcBatchTypeFunc() Code {
	return Type().Id("Batch").Op("[]").Id("baseJsonRPC").
		Line().Func().Params(Id("batch").Op("*").Id("Batch")).Id("Append").Params(Id("request").Id("baseJsonRPC")).Block(
		Op("*").Id("batch").Op("=").Append(Op("*").Id("batch"), Id("request")),
	)
}
