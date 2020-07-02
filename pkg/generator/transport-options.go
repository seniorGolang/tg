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

	srcFile.ImportName(packageFastHttp, "fasthttp")

	srcFile.Line().Type().Id("ServiceRoute").Interface(
		Id("SetRoutes").Params(Id("route").Op("*").Qual(packageFastHttpRouter, "Router")),
	)

	srcFile.Line().Type().Id("Option").Func().Params(Id("srv").Op("*").Id("Server"))
	srcFile.Type().Id("Handler").Op("=").Qual(packageFastHttp, "RequestHandler")
	srcFile.Type().Id("ErrorHandler").Func().Params(Err().Error()).Params(Error())

	srcFile.Line().Func().Id("Service").Params(Id("svc").Id("ServiceRoute")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("svc").Dot("SetRoutes").Call(Id("srv").Dot("Router").Call()),
		)),
	)

	srcFile.Line().Func().Id("AfterHTTP").Params(Id("handler").Id("Handler")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("httpAfter").Op("=").Append(Id("srv").Dot("httpAfter"), Id("handler")),
		)),
	)

	srcFile.Line().Func().Id("BeforeHTTP").Params(Id("handler").Id("Handler")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("httpBefore").Op("=").Append(Id("srv").Dot("httpBefore"), Id("handler")),
		)),
	)

	srcFile.Line().Func().Id("MaxBodySize").Params(Id("max").Int()).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
			Id("srv").Dot("maxRequestBodySize").Op("=").Id("max"),
		)),
	)
	return srcFile.Save(path.Join(outDir, "options.go"))
}
