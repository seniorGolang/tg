// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-options.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderClientCache(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	srcFile.Type().Id("cache").InterfaceFunc(func(ig *Group) {
		ig.Id("SetTTL").Params(Id(_ctx_).Qual(packageContext, "Context"), Id("key").String(), Id("value").Interface(), Id("ttl").Qual(packageTime, "Duration")).Params(Err().Error())
		ig.Id("GetTTL").Params(Id(_ctx_).Qual(packageContext, "Context"), Id("key").String(), Id("value").Interface()).Params(Id("createdAt").Qual(packageTime, "Time"), Id("ttl").Qual(packageTime, "Duration"), Err().Error())
	})

	return srcFile.Save(path.Join(outDir, "cache.go"))
}
