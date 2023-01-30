// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-jsonrpc.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr *Transport) renderClientJsonRPC(outDir string) (err error) {

	if err = pkgCopyTo("jsonrpc", outDir); err != nil {
		return err
	}
	if tr.tags.IsSet(tagClientFallback) {
		if err = pkgCopyTo("cb", outDir); err != nil {
			return err
		}
		if err = pkgCopyTo("hasher", outDir); err != nil {
			return err
		}
	}
	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)
	srcFile.ImportName(packageHttp, "http")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportAlias(packageUUID, "goUUID")
	srcFile.ImportName(packageJaegerLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportAlias(packageOpentracing, "otg")
	srcFile.ImportName(tr.tags.Value(tagPackageJSON, packageStdJSON), "json")

	srcFile.Line().Add(tr.jsonrpcClientStructFunc(outDir))
	srcFile.Line().Func().Id("New").Params(Id("endpoint").String(), Id("opts").Op("...").Id("Option")).Params(Id("cli").Op("*").Id("ClientJsonRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.List(Id("hostname"), Id("_")).Op(":=").Qual(packageOS, "Hostname").Call()
			bg.Id("cli").Op("=").Op("&").Id("ClientJsonRPC").Values(DictFunc(func(dict Dict) {
				if tr.tags.IsSet(tagClientFallback) {
					dict[Id("fallbackTTL")] = Qual(packageTime, "Hour").Op("*").Lit(24)
				}
				dict[Id("name")] = Id("hostname").Op("+").Lit("_").Op("+").Lit(tr.module.Module.Mod.String())
				dict[Id("errorDecoder")] = Id("defaultErrorDecoder")
			}))
			bg.Id("cli").Dot("applyOpts").Call(Id("opts"))
			bg.Id("cli").Dot("rpc").Op("=").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "NewClient").Call(Id("endpoint"), Id("cli").Dot("rpcOpts").Op("..."))
			if tr.tags.IsSet(tagClientFallback) {
				bg.Id("cli").Dot("cb").Op("=").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "NewCircuitBreaker").Call(Lit(tr.module.Module.Mod.String()), Id("cli").Dot("cbCfg"))
			}
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
	if tr.tags.IsSet(tagClientFallback) {
		srcFile.Line().Add(tr.jsonrpcClientProceedResponseFunc(outDir))
	}
	return srcFile.Save(path.Join(outDir, "jsonrpc.go"))
}

func (tr *Transport) jsonrpcClientProceedResponseFunc(outDir string) Code {

	return Func().
		Params(Id("cli").Op("*").Id("ClientJsonRPC")).
		Id("proceedResponse").
		Params(
			Id(_ctx_).Qual(packageContext, "Context"),
			Id("httpErr").Error(),
			Id("cacheKey").Uint64(),
			Id("fallbackCheck").Func().Params(Error()).Bool(),
			Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", tr.pkgPath(outDir)), "ResponseRPC"),
			Id("methodResponse").Interface(),
		).Params(Err().Error()).BlockFunc(func(bg *Group) {

		bg.Line()
		bg.Err().Op("=").Id("cli").Dot("cb").Dot("Execute").CallFunc(func(cg *Group) {
			cg.Func().Params().Params(Err().Error()).Block(
				If(Id("httpErr").Op("!=").Nil()).Block(
					Return(Id("httpErr")),
				),
				Return(Id("rpcResponse").Dot("GetObject").Call(Op("&").Id("methodResponse"))),
			)
			cg.Id("cb").Dot("IsSuccessful").Call(
				Func().Params(Err().Error()).Params(Id("success").Bool()).Block(
					If(Id("fallbackCheck").Op("!=").Nil()).Block(
						Return(Id("fallbackCheck")).Call(Err()),
					),
					If(Id("success").Op("=").Err().Op("==").Nil().Op(";").Id("success")).Block(
						If(Id("cli").Dot("cache").Op("!=").Nil().Op("&&").Id("cacheKey").Op("!=").Lit(0)).Block(
							Id("_").Op("=").Id("cli").Dot("cache").
								Dot("SetTTL").Call(Id(_ctx_), Qual(packageStrconv, "FormatUint").Call(Id("cacheKey"), Lit(10)), Id("methodResponse"), Id("cli").Dot("fallbackTTL")),
						),
					),
					Return(),
				),
			)
			cg.Id("cb").Dot("Fallback").Call(
				Func().Params().Params(Err().Error()).Block(
					If(Id("cli").Dot("cache").Op("!=").Nil().Op("&&").Id("cacheKey").Op("!=").Lit(0)).Block(
						List(Id("_"), Id("_"), Err()).Op("=").Id("cli").Dot("cache").Dot("GetTTL").Call(
							Id(_ctx_), Qual(packageStrconv, "FormatUint").Call(Id("cacheKey"), Lit(10)), Op("&").Id("methodResponse"),
						),
					),
					Return(),
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
		if tr.tags.IsSet(tagClientFallback) {
			sg.Line().Id("cache").Id("cache")
			sg.Line().Id("cbCfg").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "Settings")
			sg.Id("cb").Op("*").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "CircuitBreaker")
			sg.Line()
			sg.Line().Id("fallbackTTL").Qual(packageTime, "Duration")
			for _, svc := range tr.services {
				if svc.isJsonRPC() {
					sg.Id(svc.lccName() + "Fallback").Id(svc.lccName() + "Fallback")
				}
			}
		}
		sg.Line().Id("errorDecoder").Id("ErrorDecoder")
	})
}
