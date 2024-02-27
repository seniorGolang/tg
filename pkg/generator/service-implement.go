// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-implement.go at 24.06.2020, 14:11) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"
)

func (svc *service) renderImplement(outDir string) (err error) { // nolint

	outDir, _ = filepath.Abs(outDir)
	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	return srcFile.Save(path.Join(outDir, svc.lcName()+".go"))
}
