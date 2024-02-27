// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr *Transport) renderClientBatch(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(GeneratedComment())

	srcFile.ImportName(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "cb")
	srcFile.ImportName(fmt.Sprintf("%s/cache", tr.pkgPath(outDir)), "cache")
	srcFile.ImportName(fmt.Sprintf("%s/hasher", tr.pkgPath(outDir)), "hasher")
	srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "jsonrpc")

	srcFile.Line().Type().Id("RequestRPC").Struct(
		Id("retHandler").Id("rpcCallback"),
		Id("rpcRequest").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "RequestRPC"),
	)

	srcFile.Line().Type().Id("rpcCallback").Func().Params(Err().Error(), Id("response").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ResponseRPC"))

	srcFile.Line().Func().Params(Id("cli").Op("*").
		Id("ClientJsonRPC")).Id("Batch").
		Params(Id(_ctx_).Qual(packageContext, "Context"), Id("requests").Op("...").Id("RequestRPC")).BlockFunc(func(bg *Group) {
		bg.Line()
		bg.Var().Id("rpcRequests").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "RequestsRPC")
		bg.Id("callbacks").Op(":=").Make(Map(Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ID")).Id("rpcCallback"))
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
			Id("rpcRequests").Op("=").Append(Id("rpcRequests"), Id("request").Dot("rpcRequest")),
			Id("callbacks").Op("[").Id("request").Dot("rpcRequest").Dot("ID").Op("]").Op("=").Id("request").Dot("retHandler"),
		)
		bg.Var().Err().Error()
		bg.Var().Id("rpcResponses").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ResponsesRPC")
		bg.List(Id("rpcResponses"), Err()).Op("=").Id("cli").Dot("rpc").Dot("CallBatch").Call(Id(_ctx_), Id("rpcRequests"))
		bg.For(List(Id("id"), Id("response")).Op(":=").Range().Id("rpcResponses").Dot("AsMap").Call()).Block(
			If(Id("callback").Op(":=").Id("callbacks").Op("[").Id("id").Op("]").Op(";").Id("callback").Op("!=").Nil().Block(
				Id("callback").Call(Err(), Id("response")),
			)),
		)
	})
	return srcFile.Save(path.Join(outDir, "batch.go"))
}
