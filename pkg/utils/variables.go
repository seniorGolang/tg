// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (variables.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

func DictByNormalVariables(fields []types.Variable, normals []types.Variable) Dict {

	if len(fields) != len(normals) {
		panic("len of fields and normals not the same")
	}
	return DictFunc(func(d Dict) {
		for i, field := range fields {
			d[structFieldName(&field)] = Id(ToLowerCamel(normals[i].Name))
		}
	})
}

func structFieldName(field *types.Variable) *Statement {
	return Id(ToCamel(field.Name))
}
