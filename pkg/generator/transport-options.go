// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderOptions(outDir string) (err error) {

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
				If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
					Id("srv").Dot("http"+serviceName).Op("=").Id("svc"),
					Id("svc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call()),
				),
			)),
		)
	}
	srcFile.Line().Func().Id("MaxBodySize").Params(Id("max").Int()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("config").Dot("BodyLimit").Op("=").Id("max"),
		)),
	)
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
	srcFile.Line().Func().Id("Use").Params(Id("args").Op("...").Interface()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
				Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("args").Op("...")),
			),
		)),
	)
	return srcFile.Save(path.Join(outDir, "options.go"))
}
