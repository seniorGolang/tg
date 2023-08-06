// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport-errors.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr *Transport) renderRedirect(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Line().Type().Id("withRedirect").Interface(
		Id("RedirectTo").Call().String(),
	)

	return srcFile.Save(path.Join(outDir, "redirect.go"))
}
