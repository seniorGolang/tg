// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-jsonrpc.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderClientJsonRPC(outDir string) (err error) {

	if err = pkgCopyTo("jsonrpc", outDir); err != nil {
		return err
	}
	if err = pkgCopyTo("cb", outDir); err != nil {
		return err
	}
	if err = pkgCopyTo("hasher", outDir); err != nil {
		return err
	}
	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)
	srcFile.ImportName(packageHttp, "http")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportAlias(packageUUID, "goUUID")
	srcFile.ImportName(packageStdJSON, "json")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "cb")
	srcFile.ImportName(fmt.Sprintf("%s/cache", tr.pkgPath(outDir)), "cache")
	srcFile.ImportName(fmt.Sprintf("%s/hasher", tr.pkgPath(outDir)), "hasher")
	srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "jsonrpc")

	srcFile.Line().Add(tr.jsonrpcClientStructFunc(outDir))
	srcFile.Line().Func().Id("New").Params(Id("endpoint").String(), Id("opts").Op("...").Id("Option")).Params(Id("cli").Op("*").Id("ClientJsonRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.List(Id("hostname"), Id("_")).Op(":=").Qual(packageOS, "Hostname").Call()
			bg.Id("cli").Op("=").Op("&").Id("ClientJsonRPC").Values(DictFunc(func(dict Dict) {
				dict[Id("fallbackTTL")] = Qual(packageTime, "Hour").Op("*").Lit(24)
				dict[Id("name")] = Id("hostname").Op("+").Lit("_").Op("+").Lit(tr.module.Module.Mod.String())
				dict[Id("errorDecoder")] = Id("defaultErrorDecoder")
			}))
			bg.Id("cli").Dot("applyOpts").Call(Id("opts"))
			bg.Id("cli").Dot("rpc").Op("=").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "NewClient").Call(Id("endpoint"), Id("cli").Dot("rpcOpts").Op("..."))
			bg.Id("cli").Dot("cb").Op("=").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "NewCircuitBreaker").Call(Lit(tr.module.Module.Mod.String()), Id("cli").Dot("cbCfg"))
			bg.Return()
		})
	for _, name := range tr.serviceKeys() {
		svc := tr.services[name]
		if svc.tags.Contains(tagServerJsonRPC) {
			srcFile.Line().Func().Params(Id("cli").Op("*").Id("ClientJsonRPC")).Id(svc.Name).Params().Params(Op("*").Id("Client" + svc.Name)).Block(
				Return(Op("&").Id("Client" + svc.Name).Values(Dict{
					Id("ClientJsonRPC"): Id("cli"),
				})),
			)
		}
	}
	srcFile.Line().Add(tr.jsonrpcClientProceedResponseFunc(outDir))
	return srcFile.Save(path.Join(outDir, "jsonrpc.go"))
}

func (tr *Transport) jsonrpcClientProceedResponseFunc(outDir string) Code {

	return Func().
		Params(Id("cli").Op("*").Id("ClientJsonRPC")).
		Id("proceedResponse").
		Params(
			Id(_ctx_).Qual(packageContext, "Context"),
			Id("callMethod").Func().Params(Id("request").Any()).Params(Id("response").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ResponseRPC"), Err().Error()),
			Id("request").Any(),
			Id("fallbackCheck").Func().Params(Error()).Bool(),
			Id("methodResponse").Any(),
		).Params(Err().Error()).BlockFunc(func(bg *Group) {

		bg.Line()
		bg.List(Id("cacheKey"), Id("_")).Op(":=").Qual(fmt.Sprintf("%s/hasher", tr.pkgPath(outDir)), "Hash").Call(Id("request"))
		bg.Err().Op("=").Id("cli").Dot("cb").Dot("Execute").CallFunc(func(cg *Group) {
			cg.Func().Params().Params(Err().Error()).Block(
				Var().Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ResponseRPC"),
				List(Id("rpcResponse"), Err()).Op("=").Id("callMethod").Call(Id("request")),
				If(Id("rpcResponse").Op("!=").Nil().Op("&&").Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
					If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
						Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("rpcResponse").Dot("Error").Dot("Raw").Call()),
					).Else().Block(
						Err().Op("=").Qual(packageFmt, "Errorf").Call(Id("rpcResponse").Dot("Error").Dot("Message")),
					),
					Return(),
				),
				Return(Id("rpcResponse").Dot("GetObject").Call(Op("&").Id("methodResponse"))),
			)
			cg.Id("cb").Dot("IsSuccessful").Call(
				Func().Params(Err().Error()).Params(Id("success").Bool()).Block(
					If(Id("fallbackCheck").Op("!=").Nil()).Block(
						Return(Id("fallbackCheck")).Call(Err()),
					),
					If(Id("success").Op("=").Id("cli").Dot("cb").Dot("IsSuccessful").Call().Call(Err()).Op(";").Id("success")).Block(
						If(Id("cli").Dot("cache").Op("!=").Nil().Op("&&").Id("cacheKey").Op("!=").Lit(0)).Block(
							Id("_").Op("=").Id("cli").Dot("cache").
								Dot("SetTTL").Call(Id(_ctx_), Qual(packageStrconv, "FormatUint").Call(Id("cacheKey"), Lit(10)), Id("methodResponse"), Id("cli").Dot("fallbackTTL")),
						),
					),
					Return(),
				),
			)
			cg.Id("cb").Dot("Fallback").Call(
				Func().Params(Err().Error()).Params(Error()).Block(
					If(Id("cli").Dot("cache").Op("!=").Nil().Op("&&").Id("cacheKey").Op("!=").Lit(0)).Block(
						List(Id("_"), Id("_"), Err()).Op("=").Id("cli").Dot("cache").Dot("GetTTL").Call(
							Id(_ctx_), Qual(packageStrconv, "FormatUint").Call(Id("cacheKey"), Lit(10)), Op("&").Id("methodResponse"),
						),
					),
					Return(Err()),
				),
			)
		})
		bg.Return()
	})
}

func (tr *Transport) jsonrpcClientStructFunc(outDir string) Code {

	return Type().Id("ClientJsonRPC").StructFunc(func(sg *Group) {
		sg.Id("name").String()
		sg.Line().Id("rpc").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ClientRPC")
		sg.Id("rpcOpts").Op("[]").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "Option")
		sg.Line().Id("cache").Id("cache")
		sg.Line().Id("cbCfg").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "Settings")
		sg.Id("cb").Op("*").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "CircuitBreaker")
		sg.Line()
		sg.Line().Id("fallbackTTL").Qual(packageTime, "Duration")
		for _, svcName := range tr.serviceKeys() {
			svc := tr.services[svcName]
			if svc.isJsonRPC() {
				sg.Id("fallback" + svc.Name).Id("fallback" + svc.Name)
			}
		}
		sg.Line().Id("errorDecoder").Id("ErrorDecoder")
	})
}
