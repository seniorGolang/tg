// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-errors.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderErrors(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Line().Type().Id("withErrorCode").Interface(
		Id("Code").Call().Int(),
	)

	srcFile.Line().Add(tr.strErrorType())
	srcFile.Line().Add(tr.exitOnErrorFunc())

	return srcFile.Save(path.Join(outDir, "errors.go"))
}

func (tr *Transport) strErrorType() Code {

	return Type().Id("strError").String().Line().
		Func().Params(Id("e").Id("strError")).Id("Error").Params().Params(String()).Block(
		Return(String().Call(Id("e"))),
	)
}

func (tr *Transport) exitOnErrorFunc() Code {
	return Func().Id("ExitOnError").Params(Id("log").Qual(packageZeroLog, "Logger"), Err().Error(), Id("msg").String()).Block(
		If(Err().Op("!=").Nil()).Block(
			Id("log").Dot("Panic").Call().Dot("Err").Call(Err()).Dot("Msg").Call(Id("msg")),
		),
	)
}
