// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (transport-context.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"
)

func (tr Transport) renderContext(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)
	srcFile.Const().Id("CtxCancelRequest").Op("=").Lit("ctxCancelRequest")

	return srcFile.Save(path.Join(outDir, "context.go"))
}
