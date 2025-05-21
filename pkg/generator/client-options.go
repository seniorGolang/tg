// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderClientOptions(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "cb")
	srcFile.ImportName(fmt.Sprintf("%s/cache", tr.pkgPath(outDir)), "cache")
	srcFile.ImportName(fmt.Sprintf("%s/hasher", tr.pkgPath(outDir)), "hasher")
	srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "jsonrpc")

	srcFile.Line().Type().Id("Option").Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id("applyOpts").Params(Id("opts").Op("[]").Id("Option")).Block(
		For(List(Id("_"), Id("op")).Op(":=").Range().Id("opts")).Block(
			Id("op").Call(Id("cli")),
		),
	)
	srcFile.Line().Func().Id("ClientOption").Params(Id("option").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "Option")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Id("option")),
		),
	)
	srcFile.Line().Func().Id("DecodeError").Params(Id("decoder").Id("ErrorDecoder")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("errorDecoder").Op("=").Id("decoder"),
		),
	)
	srcFile.Line().Func().Id("Cache").Params(Id("cache").Id("cache")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("cache").Op("=").Id("cache"),
		),
	)
	srcFile.Line().Func().Id("CircuitBreaker").Params(Id("cfg").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "Settings")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("cbCfg").Op("=").Id("cfg"),
		),
	)
	srcFile.Line().Func().Id("FallbackTTL").Params(Id("ttl").Qual(packageTime, "Duration")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("fallbackTTL").Op("=").Id("ttl"),
		),
	)
	for _, svcName := range tr.serviceKeys() {
		svc := tr.services[svcName]
		if svc.isJsonRPC() {
			srcFile.Line().Func().Id("Fallback" + svc.Name + "Err").Params(Id("fallback").Id("fallback" + svc.Name)).Params(Id("Option")).Block(
				Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
					Id("cli").Dot("fallback" + svc.Name).Op("=").Id("fallback"),
				),
			)
		}
	}
	srcFile.Line().Func().Id("Headers").Params(Id("headers").Op("...").Interface()).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "HeaderFromCtx").Call(Id("headers").Op("..."))),
		),
	)
	srcFile.Line().Func().Id("ConfigTLS").Params(Id("tlsConfig").Op("*").Qual(packageTLS, "Config")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ConfigTLS").Call(Id("tlsConfig"))),
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
