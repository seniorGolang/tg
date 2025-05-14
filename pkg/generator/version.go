// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-rest.go at 23.06.2020, 23:36) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderVersion(outDir string, isServer bool) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Const().Id("VersionTg").Op("=").Lit(tr.version)

	if isServer {
		srcFile.Line().Add(Func().Params(Id("srv").Op("*").Id("Server")).Id("VersionTg").Params().Params(String()).Block(
			Return(Id("VersionTg")),
		))
	}
	return srcFile.Save(path.Join(outDir, "version.go"))
}
