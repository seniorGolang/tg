// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderClientError(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageStdJSON, "json")

	srcFile.Add(tr.errorJsonRPC())

	srcFile.Line().Type().Id("ErrorDecoder").Func().Params(Id("errData").Qual(packageStdJSON, "RawMessage")).Params(Error())

	srcFile.Line().Func().Id("defaultErrorDecoder").Params(Id("errData").Qual(packageStdJSON, "RawMessage")).Params(Err().Error()).Block(
		Line().Var().Id("jsonrpcError").Id("errorJsonRPC"),
		If(Err().Op("=").Qual(packageStdJSON, "Unmarshal").Call(Id("errData"), Op("&").Id("jsonrpcError")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		),
		Return(Id("jsonrpcError")),
	)

	return srcFile.Save(path.Join(outDir, "error.go"))
}
