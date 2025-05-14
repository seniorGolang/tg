// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-http.go at 25.06.2020, 11:38) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderHTTP(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.ImportName(packageFiber, "fiber")
	srcFile.ImportName(packageZeroLog, "zerolog")
	srcFile.ImportName(packageMultipart, "multipart")

	srcFile.Line().Type().Id("withRedirect").Interface(
		Id("RedirectTo").Call().String(),
	)

	srcFile.Line().Type().Id("cookieType").Interface(
		Id("Cookie").Params().Params(Op("*").Qual(packageFiber, "Cookie")),
	)

	return srcFile.Save(path.Join(outDir, "http.go"))
}
