// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-jsonrpc-client.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderClientJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportAlias(packageUUID, "goUUID")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(fmt.Sprintf("%s/cb", svc.tr.pkgPath(outDir)), "cb")
	srcFile.ImportName(fmt.Sprintf("%s/cache", svc.tr.pkgPath(outDir)), "cache")
	srcFile.ImportName(fmt.Sprintf("%s/hasher", svc.tr.pkgPath(outDir)), "hasher")
	srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "jsonrpc")

	srcFile.Line().Type().Id("Client" + svc.Name).StructFunc(func(sg *Group) {
		sg.Op("*").Id("ClientJsonRPC")
	}).Line()
	for _, method := range svc.methods {
		if method.tags.Contains(tagMethodHTTP) {
			continue
		}
		srcFile.Type().Id("ret" + svc.Name + method.Name).Op("=").Func().Params(funcDefinitionParams(ctx, method.Results))
	}
	for _, method := range svc.methods {
		if method.tags.Contains(tagMethodHTTP) {
			continue
		}
		srcFile.Line().Add(svc.jsonrpcClientMethodFunc(ctx, method, outDir))
		srcFile.Line().Add(svc.jsonrpcClientRequestFunc(ctx, method, outDir))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-jsonrpc.go"))
}

func (svc *service) jsonrpcClientMethodFunc(ctx context.Context, method *method, outDir string) Code {

	return Func().
		Params(Id("cli").Op("*").Id("Client" + svc.Name)).
		Id(method.Name).
		Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).BlockFunc(func(bg *Group) {

		bg.Line()
		bg.Id("request").Op(":=").Id(method.requestStructName()).Values(DictFunc(func(dict Dict) {
			for idx, arg := range method.fieldsArgument() {
				dict[Id(utils.ToCamel(arg.Name))] = Id(method.argsWithoutContext()[idx].Name)
			}
		}))
		bg.Var().Id("response").Id(method.responseStructName())
		if svc.tags.Contains(tagEnableClientCB) {
			bg.Var().Id("fallbackCheck").Func().Params(Error()).Bool()
			bg.If(Id("cli").Dot("fallback" + svc.Name).Op("!=").Nil()).Block(
				Id("fallbackCheck").Op("=").Id("cli").Dot("fallback" + svc.Name).Dot(method.Name),
			)
			bg.Id("callMethod").Op(":=").Func().Params(Id("request").Any()).Params(Id("response").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "ResponseRPC"), Err().Error()).Block(
				Return(Id("cli").Dot("rpc").Dot("Call").Call(Id(_ctx_), Lit(svc.lcName()+"."+method.lcName()), Id("request"))),
			)
			bg.If(Err().Op("=").
				Id("cli").Dot("proceedResponse").Call(Id(_ctx_), Id("callMethod"), Id("request"), Id("fallbackCheck"), Op("&").Id("response")).
				Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			)
		} else {
			bg.Var().Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "ResponseRPC")
			bg.If(List(Id("rpcResponse"), Err()).Op("=").Id("cli").Dot("rpc").Dot("Call").Call(Id(_ctx_), Lit(svc.lcName()+"."+method.lcName()), Id("request")).Op(";").Err().Op("!=").Nil().Op("||").Id("rpcResponse").Op("==").Nil()).Block(
				Return(),
			)
			bg.If(Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
				If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
					Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("rpcResponse").Dot("Error").Dot("Raw").Call()),
				).Else().Block(
					Err().Op("=").Qual(packageFmt, "Errorf").Call(Id("rpcResponse").Dot("Error").Dot("Message")),
				),
				Return(),
			)
			bg.If(Err().Op("=").Id("rpcResponse").Dot("GetObject").Call(Op("&").Id("response")).Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			)
		}
		bg.ReturnFunc(func(rg *Group) {
			for _, ret := range method.resultsWithoutError() {
				rg.Id("response").Dot(utils.ToCamel(ret.Name))
			}
			rg.Err()
		})
	})
}

func (svc *service) jsonrpcClientRequestFunc(ctx context.Context, method *method, outDir string) Code {

	ctxCode := Id(_ctx_).Qual(packageContext, "Context")
	return Func().Params(Id("cli").Op("*").Id("Client"+svc.Name)).
		Id("Req"+method.Name).
		Params(ctxCode, Id("callback").Id("ret"+svc.Name+method.Name), funcDefinitionParams(ctx, method.argsWithoutContext())).
		Params(Id("request").Id("RequestRPC")).BlockFunc(func(bg *Group) {

		bg.Line()
		bg.Id("request").Op("=").Id("RequestRPC").Values(Dict{
			Id("rpcRequest"): Op("&").Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "RequestRPC").Values(Dict{
				Id("ID"):      Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "NewID").Call(),
				Id("JSONRPC"): Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "Version"),
				Id("Method"):  Lit(svc.lcName() + "." + method.lcName()),
				Id("Params"): Id(method.requestStructName()).Values(DictFunc(func(dg Dict) {
					for idx, arg := range method.fieldsArgument() {
						dg[Id(utils.ToCamel(arg.Name))] = Id(method.argsWithoutContext()[idx].Name)
					}
				})),
			}),
		})
		bg.If(Id("callback").Op("!=").Nil()).Block(
			Var().Id("response").Id(method.responseStructName()),
			Id("request").Dot("retHandler").Op("=").Func().Params(
				Err().Error(),
				Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "ResponseRPC"),
			).BlockFunc(func(bg *Group) {
				if svc.tags.Contains(tagEnableClientCB) {
					bg.Var().Id("fallbackCheck").Func().Params(Error()).Bool()
					bg.If(Id("cli").Dot("fallback" + svc.Name).Op("!=").Nil()).Block(
						Id("fallbackCheck").Op("=").Id("cli").Dot("fallback" + svc.Name).Dot(method.Name),
					)
					bg.Id("callMethod").Op(":=").Func().Params(Id("request").Any()).Params(Id("response").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", svc.tr.pkgPath(outDir)), "ResponseRPC"), Err().Error()).Block(
						If(Err().Op("==").Nil().Op("&&").Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
							If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
								Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("rpcResponse").Dot("Error").Dot("Raw").Call()),
							).Else().Block(
								Err().Op("=").Qual(packageFmt, "Errorf").Call(Id("rpcResponse").Dot("Error").Dot("Message")),
							),
						).Else().Block(
							Err().Op("=").Id("rpcResponse").Dot("GetObject").Call(Op("&").Id("response")),
						),
						Return(Id("rpcResponse"), Err()),
					)
					bg.Err().Op("=").Id("cli").Dot("proceedResponse").Call(Id(_ctx_), Id("callMethod"), Id("request"), Id("fallbackCheck"), Op("&").Id("response"))
					bg.Id("callback").CallFunc(func(cg *Group) {
						for _, ret := range method.fieldsResult() {
							cg.Id("response").Dot(utils.ToCamel(ret.Name))
						}
						cg.Err()
					})
				} else {
					bg.If(Err().Op("==").Nil().Op("&&").Id("rpcResponse").Op("!=").Nil()).Block(
						If(Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
							If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
								Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("rpcResponse").Dot("Error").Dot("Raw").Call()),
							).Else().Block(
								Err().Op("=").Qual(packageFmt, "Errorf").Call(Id("rpcResponse").Dot("Error").Dot("Message")),
							),
						).Else().Block(
							Err().Op("=").Id("rpcResponse").Dot("GetObject").Call(Op("&").Id("response")),
						),
					)
					bg.Id("callback").CallFunc(func(cg *Group) {
						for _, ret := range method.fieldsResult() {
							cg.Id("response").Dot(utils.ToCamel(ret.Name))
						}
						cg.Err()
					})
				}
			}),
		)
		bg.Return()
	})
}

func (svc *service) renderClientFallbackError(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Type().Id("fallback" + svc.Name).InterfaceFunc(func(ig *Group) {
		for _, method := range svc.methods {
			ig.Id(method.Name).Params(Err().Error()).Bool()
		}
	})
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-fallback.go"))
}
