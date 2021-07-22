// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-http.go at 25.06.2020, 11:38) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr Transport) renderHTTP(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageMultipart, "multipart")

	srcFile.Line().Type().Id("cookieType").Interface(
		Id("Cookie").Params().Params(Op("*").Qual(packageFiber, "Cookie")),
	)

	srcFile.Line().Func().Id("uploadFile").Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx"), Id("key").String()).Params(Id("data").Op("[]").Byte(), Err().Error()).Block(
		Var().Id("fileHeader").Op("*").Qual(packageMultipart, "FileHeader"),
		If(List(Id("fileHeader"), Err()).Op("=").Id(_ctx_).Dot("FormFile").Call(Id("key")).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		),
		Var().Id("file").Qual(packageMultipart, "File"),
		If(List(Id("file"), Err()).Op("=").Id("fileHeader").Dot("Open").Call().Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		),
		Defer().Id("file").Dot("Close").Call(),
		Return(Qual(packageIOUtil, "ReadAll").Call(Id("file"))),
	)

	return srcFile.Save(path.Join(outDir, "http.go"))
}
