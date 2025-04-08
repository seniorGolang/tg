// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderHeader(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageErrors, "errors")
	srcFile.ImportName(packageZeroLogLog, "log")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(tr.tags.Value(tagPackageJSON, packageStdJSON), "json")

	tr.renderHeaderTypes(srcFile)
	tr.renderHeaderHandler(srcFile)
	tr.renderHeaderValue(srcFile)
	tr.renderHeaderValueInterface(srcFile)

	return srcFile.Save(path.Join(outDir, "header.go"))
}

func (tr *Transport) renderHeaderTypes(srcFile goFile) {

	srcFile.Line().Type().Id("Header").Struct(
		Id("SpanKey").String(),
		Id("SpanValue").Interface(),
		Id("RequestKey").String(),
		Id("RequestValue").Interface(),
		Id("ResponseKey").String(),
		Id("ResponseValue").Interface(),
		Id("LogKey").String(),
		Id("LogValue").Interface(),
	).Line().
		Line().Type().Id("HeaderHandler").Func().Params(Id("value").String()).Params(Id("Header"))
}

func (tr *Transport) renderHeaderHandler(srcFile goFile) {

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).
		Id("headersHandler").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).Params(Error()).BlockFunc(func(g *Group) {
		g.Line()
		g.For(List(Id("headerName"), Id("handler")).Op(":=").Range().Id("srv").Dot("headerHandlers")).Block(
			Id("value").Op(":=").Id(_ctx_).Dot("Request").Call().Dot("Header").Dot("Peek").Call(Id("headerName")),
			Id("header").Op(":=").Id("handler").Call(String().Call(Id("value"))),
			If(Id("header").Dot("RequestValue").Op("!=").Nil()).Block(
				Id(_ctx_).Dot("Request").Call().Dot("Header").Dot("Set").Call(Id("header").Dot("RequestKey"), Id("headerValue").Call(Id("header").Dot("RequestValue"))),
			),
			If(Id("header").Dot("ResponseValue").Op("!=").Nil()).Block(
				Id(_ctx_).Dot("Response").Call().Dot("Header").Dot("Set").Call(Id("header").Dot("ResponseKey"), Id("headerValue").Call(Id("header").Dot("ResponseValue"))),
			),
			If(Id("header").Dot("LogValue").Op("!=").Nil()).Block(
				Id("logger").Op(":=").Qual(packageZeroLogLog, "Ctx").Call(Id(_ctx_).Dot("UserContext").Call()).
					Dot("With").Call().Dot("Interface").Call(Id("header").Dot("LogKey"), Id("header").Dot("LogValue")).Dot("Logger").Call(),
				Id(_ctx_).Dot("SetUserContext").Call(Id("logger").Dot("WithContext").Call(Id(_ctx_).Dot("UserContext").Call())),
			),
		)
		g.Return(Id(_ctx_).Dot("Next").Call())
	})
}

func (tr *Transport) renderHeaderValue(srcFile goFile) {

	srcFile.Line().Func().Id("headerValue").Params(Id("src").Interface()).Params(Id("value").String()).Block(
		Line(),
		Switch(Id("src").Op(".").Call(Type())).Block(
			Case(String()).Return(Id("src").Op(".").Call(String())),
			Case(Id("iHeaderValue")).Return(Id("src").Op(".").Call(Id("iHeaderValue")).Dot("Header").Call()),
			Case(Qual(packageFmt, "Stringer")).Return(Id("src").Op(".").Call(Qual(packageFmt, "Stringer")).Dot("String").Call()),
			Default().List(Id("bytes"), Id("_")).Op(":=").Qual(tr.tags.Value(tagPackageJSON, packageStdJSON), "Marshal").Call(Id("src")),
			Return(String().Call(Id("bytes"))),
		),
	)
}

func (tr *Transport) renderHeaderValueInterface(srcFile goFile) {

	srcFile.Line().Type().Id("iHeaderValue").Interface(
		Id("Header").Params().Params(String()),
	)
}
