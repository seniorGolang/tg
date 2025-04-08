package generator

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) renderContext(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))

	srcFile.PackageComment(doNotEdit)

	srcFile.Add(typeMethodCallMeta())

	return srcFile.Save(path.Join(outDir, "context.go"))
}

func typeMethodCallMeta() Code {

	return Type().Id("MethodCallMeta").StructFunc(func(tg *Group) {
		tg.Id("Service").String()
		tg.Id("Method").String()
		tg.Id("Request").Any()
		tg.Id("Response").Any()
		tg.Id("Err").Error()
	})
}
