// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderOptions(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")

	srcFile.Line().Type().Id("ServiceRoute").Interface(
		Id("SetRoutes").Params(Id("route").Op("*").Qual(packageFiber, "App")),
	)

	srcFile.Line().Type().Id("Option").Func().Params(Id("srv").Op("*").Id("Server"))
	srcFile.Type().Id("Handler").Op("=").Qual(packageFiber, "Handler")
	srcFile.Type().Id("ErrorHandler").Func().Params(Err().Error()).Params(Error())

	srcFile.Line().Func().Id("Service").Params(Id("svc").Id("ServiceRoute")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
				Id("svc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call()),
			),
		)),
	)
	for _, serviceName := range tr.serviceKeys() {
		srcFile.Line().Func().Id(serviceName).Params(Id("svc").Op("*").Id("http" + serviceName)).Id("Option").Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).BlockFunc(func(gr *Group) {
					gr.Id("srv").Dot("http" + serviceName).Op("=").Id("svc")
					if tr.hasJsonRPC {
						gr.Id("svc").Dot("maxBatchSize").Op("=").Id("srv").Dot("maxBatchSize")
						gr.Id("svc").Dot("maxParallelBatch").Op("=").Id("srv").Dot("maxParallelBatch")
					}
					gr.Id("svc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call())
				}),
			)),
		)
	}
	srcFile.Line().Func().Id("SetFiberCfg").Params(Id("cfg").Qual(packageFiber, "Config")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Op("=").Id("cfg"),
			Id("srv").Dot("config").Dot("DisableStartupMessage").Op("=").True(),
		)),
	)
	srcFile.Line().Func().Id("SetReadBufferSize").Params(Id("size").Int()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Dot("ReadBufferSize").Op("=").Id("size"),
		)),
	)
	srcFile.Line().Func().Id("SetWriteBufferSize").Params(Id("size").Int()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Dot("WriteBufferSize").Op("=").Id("size"),
		)),
	)
	srcFile.Line().Func().Id("MaxBodySize").Params(Id("max").Int()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Dot("BodyLimit").Op("=").Id("max"),
		)),
	)
	if tr.hasJsonRPC {
		srcFile.Line().Func().Id("MaxBatchSize").Params(Id("size").Int()).Id("Option").Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("maxBatchSize").Op("=").Id("size"),
			)),
		)
		srcFile.Line().Func().Id("MaxBatchWorkers").Params(Id("size").Int()).Id("Option").Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("maxParallelBatch").Op("=").Id("size"),
			)),
		)
	}
	srcFile.Line().Func().Id("ReadTimeout").Params(Id("timeout").Qual(packageTime, "Duration")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Dot("ReadTimeout").Op("=").Id("timeout"),
		)),
	)
	srcFile.Line().Func().Id("WriteTimeout").Params(Id("timeout").Qual(packageTime, "Duration")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Dot("WriteTimeout").Op("=").Id("timeout"),
		)),
	)
	srcFile.Line().Func().Id("WithRequestID").Params(Id("headerName").String()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("headerHandlers").Op("[").Id("headerName").Op("]").Op("=").
				Func().Params(Id("value").String()).Params(Id("Header")).Block(
				If(Id("value").Op("==").Lit("")).Block(
					Id("value").Op("=").Qual(packageUUID, "New").Call().Dot("String").Call(),
				),
				Return(Id("Header").Block(Dict{
					Id("SpanKey"):       Lit("requestID"),
					Id("SpanValue"):     Id("value"),
					Id("ResponseKey"):   Id("headerName"),
					Id("ResponseValue"): Id("value"),
					Id("LogKey"):        Lit("requestID"),
					Id("LogValue"):      Id("value"),
				})),
			),
		)),
	)
	srcFile.Line().Func().Id("WithHeader").Params(Id("headerName").String(), Id("handler").Id("HeaderHandler")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("headerHandlers").Op("[").Id("headerName").Op("]").Op("=").Id("handler"),
		)),
	)
	srcFile.Line().Func().Id("Use").Params(Id("args").Op("...").Interface()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
				Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("args").Op("...")),
			),
		)),
	)
	return srcFile.Save(path.Join(outDir, "options.go"))
}
