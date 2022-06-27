// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-jsonrpc-client.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
	"github.com/vetcher/go-astra/types"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderClientJsonRPC(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	srcFile.ImportAlias(packageUUID, "goUUID")
	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")

	srcFile.Line().Type().Id("Client" + svc.Name).Struct(
		Op("*").Id("ClientJsonRPC"),
	).Line()

	for _, method := range svc.methods {

		if method.tags.Contains(tagMethodHTTP) {
			continue
		}
		srcFile.Type().Id("ret" + svc.Name + method.Name).Func().Params(funcDefinitionParams(ctx, method.Results))
	}

	for _, method := range svc.methods {

		if method.tags.Contains(tagMethodHTTP) {
			continue
		}
		srcFile.Line().Add(svc.jsonrpcClientRequestFunc(ctx, method))
		srcFile.Line().Add(svc.jsonrpcClientMethodFunc(ctx, method))
	}

	return srcFile.Save(path.Join(outDir, svc.lcName()+"-jsonrpc.go"))
}

func (svc *service) jsonrpcClientRequestFunc(ctx context.Context, method *method) Code {

	return Func().Params(Id("cli").Op("*").Id("Client"+svc.Name)).Id("Req"+method.Name).Params(Id("ret").Id("ret"+svc.Name+method.Name), funcDefinitionParams(ctx, method.argsWithoutContext())).Params(Id("request").Id("baseJsonRPC")).Block(

		Line().Id("request").Op("=").Id("baseJsonRPC").Values(Dict{
			Id("Version"): Id("Version"),
			Id("Method"):  Lit(svc.lcName() + "." + method.lcName()),
			Id("Params"): Id("request" + svc.Name + method.Name).Values(DictFunc(func(d Dict) {
				for _, arg := range method.argsWithoutContext() {
					d[Id(utils.ToCamel(arg.Name))] = Id(arg.Name)
				}
			})),
		}),

		Var().Err().Error(),
		Var().Id("response").Id(method.responseStructName()),

		Line().If(Id("ret").Op("!=").Nil()).Block(
			Id("request").Dot("retHandler").Op("=").Func().Params(Id("jsonrpcResponse").Id("baseJsonRPC")).Block(
				If(Id("jsonrpcResponse").Dot("Error").Op("!=").Nil()).Block(
					Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("jsonrpcResponse").Dot("Error")),
					Id("ret").CallFunc(func(cg *Group) {
						for _, ret := range method.resultsWithoutError() {
							cg.Id("response").Dot(utils.ToCamel(ret.Name))
						}
						cg.Err()
					}),
					Return(),
				),
				Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id("jsonrpcResponse").Dot("Result"), Op("&").Id("response")),
				Id("ret").CallFunc(func(cg *Group) {
					for _, ret := range method.resultsWithoutError() {
						cg.Id("response").Dot(utils.ToCamel(ret.Name))
					}
					cg.Err()
				}),
			),
			Id("request").Dot("ID").Op("=").Op("[]").Byte().Call(Lit(`"`).Op("+").Qual(packageUUID, "New").Call().Dot("String").Call().Op("+").Lit(`"`)),
		),
		Return(),
	)
}

func (svc *service) jsonrpcClientMethodFunc(ctx context.Context, method *method) Code {

	return Func().Params(Id("cli").Op("*").Id("Client"+svc.Name)).Id(method.Name).Params(funcDefinitionParams(ctx, method.Args)).Params(funcDefinitionParams(ctx, method.Results)).Block(

		Line().Id("retHandler").Op(":=").Func().ParamsFunc(func(pg *Group) {
			for _, ret := range method.Results {
				pg.Id("_" + ret.Name).Add(fieldType(context.Background(), ret.Type, true))
			}
		}).BlockFunc(func(bg *Group) {
			for _, ret := range method.Results {
				bg.Id(ret.Name).Op("=").Id("_" + ret.Name)
			}
		}),

		If(Id("blockErr").Op(":=").Id("cli").Dot("Batch").Call(Id(_ctx_), Id("cli").Dot("Req"+method.Name).CallFunc(func(cg *Group) {
			cg.Id("retHandler")
			for _, arg := range method.argsWithoutContext() {

				argCode := Id(arg.Name)

				if types.IsEllipsis(arg.Type) {
					argCode.Op("...")
				}
				cg.Add(argCode)
			}
		})).Op(";").Id("blockErr").Op("!=").Nil()).Block(
			Err().Op("=").Id("blockErr"),
			Return(),
		),
		Return(),
	)
}
