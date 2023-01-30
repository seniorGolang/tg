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

func (tr *Transport) renderClientOptions(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Line().Type().Id("Option").Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("applyOpts").Params(Id("opts").Op("[]").Id("Option")).Block(
		For(List(Id("_"), Id("op")).Op(":=").Range().Id("opts")).Block(
			Id("op").Call(Id("cli")),
		),
	)

	srcFile.Line().Func().Id("DecodeError").Params(Id("decoder").Id("ErrorDecoder")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("errorDecoder").Op("=").Id("decoder"),
		),
	)
	if tr.tags.IsSet(tagClientFallback) {
		srcFile.Line().Func().Id("Cache").Params(Id("cache").Id("cache")).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
				Id("cli").Dot("cache").Op("=").Id("cache"),
			),
		)
		srcFile.Line().Func().Id("FallbackTTL").Params(Id("ttl").Qual(packageTime, "Duration")).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
				Id("cli").Dot("fallbackTTL").Op("=").Id("ttl"),
			),
		)
	}
	srcFile.Line().Func().Id("Headers").Params(Id("headers").Op("...").Interface()).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "HeaderFromCtx").Call(Id("headers").Op("..."))),
		),
	)
	srcFile.Line().Func().Id("Insecure").Params().Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "Insecure").Call()),
		),
	)
	srcFile.Line().Func().Id("LogRequest").Params().Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "LogRequest").Call()),
		),
	)
	srcFile.Line().Func().Id("LogOnError").Params().Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "LogOnError").Call()),
		),
	)
	return srcFile.Save(path.Join(outDir, "options.go"))
}
