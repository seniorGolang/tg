// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (client-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderClientOptions(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Const().Id("headerRequestID").Op("=").Lit("X-Request-Id")

	srcFile.Line().Type().Id("Option").Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))

	srcFile.Line().Func().Id("DecodeError").Params(Id("decoder").Id("ErrorDecoder")).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("errorDecoder").Op("=").Id("decoder"),
		),
	)
	srcFile.Line().Func().Id("Headers").Params(Id("headers").Op("...").String()).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("ClientJsonRPC"))).Block(
			Id("cli").Dot("headers").Op("=").Id("headers"),
		),
	)
	return srcFile.Save(path.Join(outDir, "options.go"))
}
