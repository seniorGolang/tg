// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (utils.go at 09.06.2020, 2:09) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
	"github.com/pkg/errors"

	"github.com/seniorGolang/tg/v2/pkg/astra"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	"github.com/seniorGolang/tg/v2/pkg/mod"
	"github.com/seniorGolang/tg/v2/pkg/utils"
)

type pair[K, V any] struct {
	Key   K
	Value V
}

func newPair[K, V any](k K, v V) pair[K, V] {
	return pair[K, V]{
		Key:   k,
		Value: v,
	}
}

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

func searchType(pkg, name string) (retType types.Type) {

	if retType, _ = parseType(pkg, name); retType == nil {
		pkgPath := mod.PkgModPath(pkg)
		if retType, _ = parseType(pkgPath, name); retType == nil {
			pkgPath = path.Join("./vendor", pkg)
			if retType, _ = parseType(pkgPath, name); retType == nil {
				pkgPath = trimLocalPkg(pkg)
				retType, _ = parseType(pkgPath, name)
			}
		}
	}
	return
}

func isPointerType(v types.Type) (isPointer bool) {

	_, isPointer = v.(types.TPointer)
	return
}

func nestedType(field types.Type, pkg string, path []string) (nested types.Type) {

	if len(path) == 0 {
		return field
	}
	switch f := field.(type) {
	case types.TImport:
		return nestedType(f.Next, f.Import.Package, path)
	case types.TName:
		if nextType := searchType(pkg, f.TypeName); nextType != nil {
			return nestedType(nextType, pkg, path[1:])
		}
		return f
	case types.Struct:
		for _, field := range f.Fields {
			if field.Name == path[0] {
				return nestedType(field.Type, pkg, path[1:])
			}
		}
	case types.TArray:
		return f
	case types.TMap:
		return f
	case types.TPointer:
		return nestedType(f.Next, pkg, path[1:])
	case types.TInterface:
		return f
	case types.TEllipsis:
		return f
	default:
		return f
	}
	return
}

func structField(ctx context.Context, field types.StructField, template string) *Statement {

	var isInlined bool
	tags := map[string]string{"json": fmt.Sprintf(template, field.Name)}
	for tag, values := range field.Tags {
		if tag == "json" {
			for _, value := range values {
				if value == "inline" {
					isInlined = true
				}
			}
		}
		tags[tag] = strings.Join(values, ",")
	}
	var s *Statement
	if isInlined {
		s = fieldType(ctx, field.Type, false)
		s.Tag(map[string]string{"json": ",inline"})
	} else {
		s = Id(utils.ToCamel(field.Name))
		s.Add(fieldType(ctx, field.Type, false))
		s.Tag(tags)
	}
	if types.IsEllipsis(field.Type) {
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
				if srcFile, ok := ctx.Value("code").(goFile); ok {
					if strings.HasSuffix(f.Import.Package, f.Import.Name) {
						srcFile.ImportName(f.Import.Package, f.Import.Name)
					} else {
						srcFile.ImportAlias(f.Import.Package, f.Import.Name)
					}
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

	var list = make([]Code, 0, len(fields))
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

	var list = make([]Code, 0, len(fields))
	for _, field := range fields {
		v := Id(object).Dot(utils.ToCamel(field.Name))
		if types.IsEllipsis(field.Type) {
			v.Op("...")
		}
		list = append(list, v)
	}
	return List(list...)
}

func (ts *clientTS) searchType(pkg, name string) (retType types.Type, constants []types.Constant) {

	if retType, constants = parseType(pkg, name); retType == nil {
		pkgPath := mod.PkgModPath(pkg)
		if retType, constants = parseType(pkgPath, name); retType == nil {
			pkgPath = path.Join("./vendor", pkg)
			if retType, constants = parseType(pkgPath, name); retType == nil {
				pkgPath = trimLocalPkg(pkg)
				retType, constants = parseType(pkgPath, name)
			}
		}
	}
	return
}

func trimLocalPkg(pkg string) (pgkPath string) {

	module := getModName()

	if module == "" {
		return pkg
	}

	moduleTokens := strings.Split(module, "/")
	pkgTokens := strings.Split(pkg, "/")

	if len(pkgTokens) < len(moduleTokens) {
		return pkg
	}

	pgkPath = path.Join(strings.Join(pkgTokens[len(moduleTokens):], "/"))
	return
}

func getModName() (module string) {

	modPath, _ := mod.GoModPath(".")
	modFile, err := os.OpenFile(modPath, os.O_RDONLY, os.ModePerm)

	if err != nil {
		return
	}
	defer modFile.Close()

	rd := bufio.NewReader(modFile)
	if module, err = rd.ReadString('\n'); err != nil {
		return ""
	}
	module = strings.Trim(module, "\n")

	moduleTokens := strings.Split(module, " ")

	if len(moduleTokens) == 2 {
		module = strings.TrimSpace(moduleTokens[1])
	}
	return
}

func parseType(relPath, name string) (retType types.Type, constants []types.Constant) {

	GoProjectPath := mod.GoProjectPath(".")
	pkgPath := path.Join(GoProjectPath, relPath)
	_ = filepath.Walk(pkgPath, func(filePath string, info os.FileInfo, err error) (retErr error) {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		var srcFile *types.File
		if srcFile, err = astra.ParseFile(filePath, astra.IgnoreMethods); err != nil {
			if retErr = errors.Wrap(err, fmt.Sprintf("%s,%s", relPath, name)); retErr != nil {
				return
			}
			return err
		}
		constants = append(constants, srcFile.Constants...)
		for _, typeInfo := range srcFile.Interfaces {
			if typeInfo.Name == name {
				retType = types.TInterface{Interface: &typeInfo}
				return
			}
		}
		for _, typeInfo := range srcFile.Types {
			if typeInfo.Name == name {
				retType = typeInfo.Type
				return
			}
		}
		for _, structInfo := range srcFile.Structures {
			if structInfo.Name == name {
				retType = structInfo
				return
			}
		}
		return
	})
	return
}
