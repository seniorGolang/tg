// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (utils.go at 09.06.2020, 2:09) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/vetcher/go-astra/types"

	"github.com/seniorGolang/tg/pkg/utils"
)

func removeSkippedFields(fields []types.Variable, skipFields []string) []types.Variable {

	var result []types.Variable

	for _, field := range fields {
		add := true
		for _, skip := range skipFields {
			if strings.TrimSpace(skip) == field.Name {
				add = false
				break
			}
		}
		if add {
			result = append(result, field)
		}
	}
	return result
}

func isContextFirst(fields []types.Variable) bool {
	if len(fields) == 0 {
		return false
	}
	name := types.TypeName(fields[0].Type)
	return name != nil &&
		types.TypeImport(fields[0].Type) != nil &&
		types.TypeImport(fields[0].Type).Package == packageContext && *name == "Context"
}

func isErrorLast(fields []types.Variable) bool {
	if len(fields) == 0 {
		return false
	}
	name := types.TypeName(fields[len(fields)-1].Type)
	return name != nil &&
		types.TypeImport(fields[len(fields)-1].Type) == nil &&
		*name == "error"
}

func structField(ctx context.Context, field types.StructField) *Statement {

	s := Id(utils.ToCamel(field.Name))

	s.Add(fieldType(ctx, field.Variable.Type, false))

	tags := map[string]string{"json": field.Name}

	for tag, values := range field.Tags {
		tags[tag] = strings.Join(values, ",")
	}
	s.Tag(tags)

	if types.IsEllipsis(field.Variable.Type) {
		s.Comment("This field was defined with ellipsis (...).")
	}
	return s
}

func fieldType(ctx context.Context, field types.Type, allowEllipsis bool) *Statement {

	c := &Statement{}

	imported := false

	for field != nil {
		switch f := field.(type) {
		case types.TImport:
			if f.Import != nil {
				if srcFile, ok := ctx.Value("code").(srcFile); ok {
					srcFile.ImportName(f.Import.Package, f.Import.Base.Name)
					c.Qual(f.Import.Package, "")
				} else {
					c.Qual(f.Import.Package, "")
				}
				imported = true
			}
			field = f.Next
		case types.TName:
			if !imported && !types.IsBuiltin(f) {
			} else {
				c.Id(f.TypeName)
			}
			field = nil
		case types.TArray:
			if f.IsSlice {
				c.Index()
			} else if f.ArrayLen > 0 {
				c.Index(Lit(f.ArrayLen))
			}
			field = f.Next
		case types.TMap:
			return c.Map(fieldType(ctx, f.Key, false)).Add(fieldType(ctx, f.Value, false))
		case types.TPointer:
			c.Op("*")
			field = f.Next
		case types.TInterface:
			mhds := interfaceType(ctx, f.Interface)
			return c.Interface(mhds...)
		case types.TEllipsis:
			if allowEllipsis {
				c.Op("...")
			} else {
				c.Index()
			}
			field = f.Next
		default:
			return c
		}
	}
	return c
}

func interfaceType(ctx context.Context, p *types.Interface) (code []Code) {
	for _, x := range p.Methods {
		code = append(code, functionDefinition(ctx, x))
	}
	return
}

func functionDefinition(ctx context.Context, signature *types.Function) *Statement {
	return Id(signature.Name).
		Params(funcDefinitionParams(ctx, signature.Args)).
		Params(funcDefinitionParams(ctx, signature.Results))
}

func funcDefinitionParams(ctx context.Context, fields []types.Variable) *Statement {
	c := &Statement{}
	c.ListFunc(func(g *Group) {
		for _, field := range fields {
			g.Id(utils.ToLowerCamel(field.Name)).Add(fieldType(ctx, field.Type, true))
		}
	})
	return c
}

func paramNames(fields []types.Variable) *Statement {
	var list []Code
	for _, field := range fields {
		v := Id(utils.ToLowerCamel(field.Name))
		if types.IsEllipsis(field.Type) {
			v.Op("...")
		}
		list = append(list, v)
	}
	return List(list...)
}

func callParamNames(object string, fields []types.Variable) *Statement {
	var list []Code
	for _, field := range fields {
		v := Id(object).Dot(utils.ToCamel(field.Name))
		if types.IsEllipsis(field.Type) {
			v.Op("...")
		}
		list = append(list, v)
	}
	return List(list...)
}
