// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (batch.go) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

type batchTarget struct {
	receiver     string
	receiverType string
}

func batchJobType() (code Code) {

	return Type().Id("batchJob").Struct(
		Id("idx").Int(),
		Id("request").Id("baseJsonRPC"),
	)
}

func batchDoFunc(target batchTarget) (code Code) {

	recv := func() *Statement { return Id(target.receiver) }

	return Func().Params(recv().Op("*").Id(target.receiverType)).Id("doBatch").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("requests").Op("[]").Id("baseJsonRPC")).
		Params(Id("responses").Id("jsonrpcResponses")).
		BlockFunc(func(bg *Group) {

			bg.If(Len(Id("requests")).Op(">").Add(recv().Dot("maxBatchSize"))).Block(
				Id("responses").Dot("append").Call(Id("makeErrorResponseJsonRPC").Call(Nil(), Id("invalidRequestError"), Lit("batch size exceeded"), Nil())),
				Return(),
			)
			bg.Id("userCtx").Op(":=").Id(_ctx_).Dot("UserContext").Call()
			bg.If(Qual(packageStrings, "EqualFold").Call(Id(_ctx_).Dot("Get").Call(Lit(syncHeader)), Lit("true"))).Block(
				Id("results").Op(":=").Make(Index().Op("*").Id("baseJsonRPC"), Len(Id("requests"))),
				For(List(Id("idx"), Id("request")).Op(":=").Range().Id("requests")).Block(
					Id("response").Op(":=").Add(recv().Dot("doSingleBatch").Call(Id("userCtx"), Id(_ctx_), Id("request"))),
					If(Id("request").Dot("ID").Op("!=").Nil()).Block(
						Id("results").Index(Id("idx")).Op("=").Id("response"),
					),
				),
				Id("responses").Op("=").Make(Id("jsonrpcResponses"), Lit(0), Len(Id("requests"))),
				For(List(Id("_"), Id("response")).Op(":=").Range().Id("results")).Block(
					Id("responses").Dot("append").Call(Id("response")),
				),
				Return(),
			)
			bg.Var().Id("wg").Qual(packageSync, "WaitGroup")
			bg.Id("workers").Op(":=").Add(recv().Dot("maxParallelBatch"))
			bg.If(Len(Id("requests")).Op("<").Id("workers")).Block(
				Id("workers").Op("=").Len(Id("requests")),
			)
			bg.Id("results").Op(":=").Make(Index().Op("*").Id("baseJsonRPC"), Len(Id("requests")))
			bg.Id("jobs").Op(":=").Make(Chan().Id("batchJob"), Id("workers"))
			bg.For(Id("i").Op(":=").Lit(0).Op(";").Id("i").Op("<").Id("workers").Op(";").Id("i").Op("++")).Block(
				Id("wg").Dot("Add").Call(Lit(1)),
				Go().Func().Params().Block(
					Defer().Id("wg").Dot("Done").Call(),
					For(Id("job").Op(":=").Range().Id("jobs")).Block(
						Id("response").Op(":=").Add(recv().Dot("doSingleBatch").Call(Id("userCtx"), Nil(), Id("job").Dot("request"))),
						If(Id("job").Dot("request").Dot("ID").Op("!=").Nil()).Block(
							Id("results").Index(Id("job").Dot("idx")).Op("=").Id("response"),
						),
					),
				).Call(),
			)
			bg.For(List(Id("idx"), Id("request")).Op(":=").Range().Id("requests")).Block(
				Id("jobs").Op("<-").Id("batchJob").Values(Dict{
					Id("idx"):     Id("idx"),
					Id("request"): Id("request"),
				}),
			)
			bg.Close(Id("jobs"))
			bg.Id("wg").Dot("Wait").Call()
			bg.Id("responses").Op("=").Make(Id("jsonrpcResponses"), Lit(0), Len(Id("requests")))
			bg.For(List(Id("_"), Id("response")).Op(":=").Range().Id("results")).Block(
				Id("responses").Dot("append").Call(Id("response")),
			)
			bg.Return()
		})
}
