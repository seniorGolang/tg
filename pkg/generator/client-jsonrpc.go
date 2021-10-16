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

	srcFile.ImportName(packageJson, "json")
	srcFile.ImportName(packageHttp, "http")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageJaegerLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportAlias(packageOpentracing, "otg")
	srcFile.ImportAlias(packageUUID, "goUUID")

	srcFile.Line().Add(tr.jsonrpcConstants(true))

	srcFile.Line().Add(tr.idJsonRPC())
	srcFile.Type().Id("ErrorDecoder").Func().Params(Id("errData").Qual(packageJson, "RawMessage")).Params(Error())
	srcFile.Line().Add(tr.baseJsonRPC(true))
	srcFile.Line().Add(tr.errorJsonRPC())
	srcFile.Line().Add(tr.jsonrpcClientStructFunc())
	srcFile.Line().Add(tr.jsonrpcBatchTypeFunc())

	srcFile.Line().Func().Id("New").Params(Id("name").String(), Id("log").Qual(packageZeroLog, "Logger"), Id("url").String(), Id("opts").Op("...").Id("Option")).Params(Id("cli").Op("*").Id("ClientJsonRPC")).Block(

		Id("cli").Op("=").Op("&").Id("ClientJsonRPC").Values(Dict{
			Id("name"):         Id("name"),
			Id("log"):          Id("log"),
			Id("url"):          Id("url"),
			Id("errorDecoder"): Id("defaultErrorDecoder"),
		}),

		Line().For(List(Id("_"), Id("opt")).Op(":=").Range().Id("opts")).Block(
			Id("opt").Call(Id("cli")),
		),
		Return(),
	)

	var hasTrace bool
	for _, name := range tr.serviceKeys() {
		svc := tr.services[name]
		if svc.tags.IsSet(tagTrace) {
			hasTrace = true
		}
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
	srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("Batch").
		Params(Id(_ctx_).Qual(packageContext, "Context"), Id("requests").Op("...").Id("baseJsonRPC")).Params(Err().Error()).BlockFunc(func(pg *Group) {
		if hasTrace {
			pg.Id("span").Op(":=").Id("extractSpan").Call(Id("cli").Dot("log"), Id(_ctx_), Id("cli").Dot("name"))
			pg.Return(Id("cli").Dot("jsonrpcCall").Call(Id(_ctx_), Id("cli").Dot("log"), Id("span"), Id("requests").Op("...")))
		} else {
			pg.Return(Id("cli").Dot("jsonrpcCall").Call(Id(_ctx_), Id("cli").Dot("log"), Id("requests").Op("...")))
		}
	})
	srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("BatchFunc").Params(Id(_ctx_).Qual(packageContext, "Context"), Id("batchFunc").Func().
		Params(Id("requests").Op("*").Id("Batch"))).Params(Err().Error()).BlockFunc(func(pg *Group) {
		pg.Var().Id("requests").Id("Batch")
		pg.Id("batchFunc").Call(Op("&").Id("requests"))
		if hasTrace {
			pg.Id("span").Op(":=").Id("extractSpan").Call(Id("cli").Dot("log"), Id(_ctx_), Id("cli").Dot("name"))
			pg.Return(Id("cli").Dot("jsonrpcCall").Call(Id(_ctx_), Id("cli").Dot("log"), Id("span"), Id("requests").Op("...")))
		} else {
			pg.Return(Id("cli").Dot("jsonrpcCall").Call(Id(_ctx_), Id("cli").Dot("log"), Id("requests").Op("...")))
		}
	})
	srcFile.Line().Add(tr.jsonrpcClientCallFunc(hasTrace))
	return srcFile.Save(path.Join(outDir, "jsonrpc.go"))
}

func (tr Transport) jsonrpcClientStructFunc() Code {

	return Type().Id("ClientJsonRPC").Struct(
		Id("url").String(),
		Id("name").String(),
		Id("log").Qual(packageZeroLog, "Logger"),
		Id("headers").Op("[]").String(),
		Line().Id("errorDecoder").Id("ErrorDecoder"),
	)
}

func (tr Transport) jsonrpcClientCallFunc(hasTrace bool) Code {

	return Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("jsonrpcCall").
		ParamsFunc(func(pg *Group) {
			pg.Id(_ctx_).Qual(packageContext, "Context")
			pg.Id("log").Qual(packageZeroLog, "Logger")
			if hasTrace {
				pg.Id("span").Qual(packageOpentracing, "Span")
			}
			pg.Id("requests").Op("...").Id("baseJsonRPC")
		}).Params(Err().Error()).BlockFunc(func(bg *Group) {
		if hasTrace {
			bg.Defer().Id("span").Dot("Finish").Call()
		}
		bg.Id("agent").Op(":=").Qual(packageFiber, "AcquireAgent").Call()
		bg.Id("req").Op(":=").Id("agent").Dot("Request").Call()
		bg.Id("resp").Op(":=").Qual(packageFiber, "AcquireResponse").Call()
		bg.Id("agent").Dot("SetResponse").Call(Id("resp"))
		bg.Defer().Qual(packageFiber, "ReleaseResponse").Call(Id("resp"))

		bg.Line().Id("req").Dot("SetRequestURI").Call(Id("cli").Dot("url"))
		bg.Id("agent").Dot("ContentType").Call(Id("contentTypeJson"))
		bg.Id("req").Dot("Header").Dot("SetMethod").Call(Qual(packageFiber, "MethodPost"))
		bg.If(Err().Op("=").Id("agent").Dot("Parse").Call().Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		)

		bg.Line().List(Id("requestID"), Id("_")).Op(":=").Id(_ctx_).Dot("Value").Call(Id("headerRequestID")).Op(".(").String().Op(")")
		bg.If(Id("requestID").Op("==").Lit("")).Block(
			Id("requestID").Op("=").Qual(packageUUID, "New").Call().Dot("String").Call(),
		)
		bg.Id("req").Dot("Header").Dot("Set").Call(Id("headerRequestID"), Id("requestID"))
		bg.For(List(Id("_"), Id("header")).Op(":=").Range().Id("cli").Dot("headers")).Block(
			If(List(Id("value"), Id("ok")).Op(":=").Id(_ctx_).Dot("Value").Call(Id("header")).Op(".(").String().Op(")")).Op(";").Id("ok").Block(
				Id("req").Dot("Header").Dot("Set").Call(Id("header"), Id("value")),
			),
		)
		bg.If(Err().Op("=").Qual(packageJson, "NewEncoder").Call(Id("req").Dot("BodyWriter").Call()).Dot("Encode").Call(Id("requests")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		)
		if hasTrace {
			bg.Id("injectSpan").Call(Id("log"), Id("span"), Id("req"))
		}
		bg.If(Err().Op("=").Id("agent").Dot("Do").Call(Id("req"), Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		)
		bg.Id("responseMap").Op(":=").Make(Map(String()).Func().Params(Id("baseJsonRPC")))
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(

			If(Id("request").Dot("ID").Op("!=").Nil()).Block(
				Id("responseMap").Op("[").String().Call(Id("request").Dot("ID")).Op("]").Op("=").Id("request").Dot("retHandler"),
			),
		)
		bg.Var().Id("responses").Op("[]").Id("baseJsonRPC")
		bg.If(Err().Op("=").Qual(packageJson, "Unmarshal").Call(Id("resp").Dot("Body").Call(), Op("&").Id("responses")).Op(";").Err().Op("!=").Nil()).Block(
			Id("cli").Dot("log").Dot("Error").Call().Dot("Err").Call(Err()).Dot("Str").Call(Lit("response"), String().Call(Id("resp").Dot("Body").Call())).Dot("Msg").Call(Lit("unmarshal response error")),
			Return(),
		)
		bg.For(List(Id("_"), Id("response")).Op(":=").Range().Id("responses")).Block(
			If(List(Id("handler"), Id("found")).Op(":=").Id("responseMap").Op("[").String().Call(Id("response").Dot("ID")).Op("]").Op(";").Id("found")).Block(
				Id("handler").Call(Id("response")),
			),
		)
		bg.Return()
	})
}

func (tr Transport) jsonrpcBatchTypeFunc() Code {
	return Type().Id("Batch").Op("[]").Id("baseJsonRPC").
		Line().Func().Params(Id("batch").Op("*").Id("Batch")).Id("Append").Params(Id("request").Id("baseJsonRPC")).Block(
		Op("*").Id("batch").Op("=").Append(Op("*").Id("batch"), Id("request")),
	)
}
