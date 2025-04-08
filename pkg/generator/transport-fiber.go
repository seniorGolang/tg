// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderFiber(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageErrors, "errors")
	srcFile.ImportName(packageZeroLog, "zerolog")

	srcFile.Line().Const().Id("logLevelHeader").Op("=").Lit("X-Log-Level")

	tr.renderFiberLogger(srcFile)
	tr.logLevelHandler(srcFile)
	tr.renderFiberRecover(srcFile)

	return srcFile.Save(path.Join(outDir, "fiber.go"))
}

func (tr *Transport) renderFiberLogger(srcFile goFile) {

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).Id("setLogger").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Error().Block(
		Id(_ctx_).Dot("SetUserContext").Call(Id("srv").Dot("log").Dot("WithContext").Call(Id(_ctx_).Dot("UserContext").Call())),
		Return(Id(_ctx_).Dot("Next").Call()),
	)
}

func (tr *Transport) logLevelHandler(srcFile goFile) {

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).Id("logLevelHandler").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Error().Block(

		Line().
			If(Id("levelName").Op(":=").String().Call(Id(_ctx_).
				Dot("Request").Call().Dot("Header").
				Dot("Peek").Call(Id("logLevelHeader"))).Op(";").Id("levelName").Op("!=").Lit("")).Block(
			If(List(Id("level"), Err()).Op(":=").Qual(packageZeroLog, "ParseLevel").Call(Id("levelName")).Op(";").Err().Op("==").Nil()).Block(
				Id("logger").Op(":=").Qual(packageZeroLogLog, "Ctx").Call(Id(_ctx_).Dot("UserContext").Call()).Dot("Level").Call(Id("level")),
				Id(_ctx_).Dot("SetUserContext").Call(Id("logger").Dot("WithContext").Call(Id(_ctx_).Dot("UserContext").Call())),
			),
		),
		Return(Id(_ctx_).Dot("Next").Call()),
	)
}

func (tr *Transport) renderFiberRecover(srcFile goFile) {

	srcFile.Line().Func().Id("recoverHandler").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Error().Block(
		Defer().Func().Params().Block(
			If(Id("r").Op(":=").Recover().Op(";").Id("r").Op("!=").Nil().Block(
				List(Err(), Id("ok")).Op(":=").Id("r").Op(".").Call(Error()),
				If(Op("!").Id("ok")).Block(
					Err().Op("=").Qual(packageErrors, "New").Call(Qual(packageFmt, "Sprintf").Call(Lit("%v"), Id("r"))),
				),
				Qual(packageZeroLogLog, "Ctx").Call(Id(_ctx_).Dot("UserContext").Call()).Dot("Error").Call().Dot("Stack").Call().Dot("Err").Call(Qual(packageErrors, "Wrap").Call(Err(), Lit("recover"))).
					Dot("Str").Call(Lit("method"), Id(_ctx_).Dot("Method").Call()).
					Dot("Str").Call(Lit("path"), Id(_ctx_).Dot("OriginalURL").Call()).
					Dot("Msg").Call(Lit("panic occurred")),
				Id(_ctx_).Dot("Status").Call(Qual(packageFiber, "StatusInternalServerError")),
			)),
		).Call(),
		Return(Id(_ctx_).Dot("Next").Call()),
	)
}
