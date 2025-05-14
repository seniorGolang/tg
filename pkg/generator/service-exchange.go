// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-exchange.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

func (svc *service) renderExchange(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	for _, method := range svc.methods {
		srcFile.Add(svc.exchange(ctx, method.requestStructName(), method.fieldsArgument())).Line()
		srcFile.Add(svc.exchange(ctx, method.responseStructName(), method.fieldsResult())).Line()
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-exchange.go"))
}

func (svc *service) exchange(ctx context.Context, name string, params []types.StructField) Code {

	if len(params) == 0 {
		return Comment("Formal exchange type, please do not delete.").Line().Type().Id(name).Struct()
	}
	template := "%s,omitempty"
	if svc.tags.IsSet(tagDisableOmitEmpty) {
		template = "%s"
	}
	return Type().Id(name).StructFunc(func(g *Group) {
		for _, param := range params {
			g.Add(structField(ctx, param, template))
		}
	})
}
