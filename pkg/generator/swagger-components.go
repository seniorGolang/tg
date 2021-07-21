// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (swagger-components.go at 24.06.2020, 0:35) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/vetcher/go-astra"
	"github.com/vetcher/go-astra/types"

	"github.com/seniorGolang/tg/pkg/mod"
	"github.com/seniorGolang/tg/pkg/tags"
	"github.com/seniorGolang/tg/pkg/utils"
)

func (doc *swagger) registerStruct(name, pkgPath string, mTags tags.DocTags, fields []types.StructField) {

	if len(fields) == 0 {
		doc.schemas[name] = swSchema{Type: "object"}
		return
	}

	if doc.schemas == nil {
		doc.schemas = make(swSchemas)
	}

	structType := types.Struct{
		Base: types.Base{Name: name, Docs: mTags.ToDocs()},
	}

	for _, field := range fields {
		field.Base.Docs = mTags.Sub(utils.ToLowerCamel(field.Name)).ToDocs()
		structType.Fields = append(structType.Fields, field)
	}
	doc.schemas[name] = doc.walkVariable(name, pkgPath, structType, mTags)
}

func (doc *swagger) registerComponents(typeName, pkgPath string, varType types.Type) {

	if doc.schemas == nil {
		doc.schemas = make(swSchemas)
	}
	doc.schemas[typeName] = doc.walkVariable(typeName, pkgPath, varType, nil)
}

func (doc *swagger) walkVariable(typeName, pkgPath string, varType types.Type, varTags tags.DocTags) (schema swSchema) {

	if len(varTags) > 0 {

		schema.Description = varTags.Value(tagDesc)
		if example := varTags.Value(tagExample); example != "" {

			var value interface{} = example
			_ = json.Unmarshal([]byte(example), &value)
			schema.Example = value
		}
		if format := varTags.Value(tagFormat); format != "" {
			schema.Format = format
		}
		if newType := varTags.Value(tagType); newType != "" {
			schema.Type = newType
			return
		}
	}

	if newType, format := castType(varType.String()); newType != varType.String() {

		schema.Type = newType
		schema.Format = format
		return
	}

	switch vType := varType.(type) {

	case types.TName:

		if types.IsBuiltin(varType) {
			schema.Type = vType.TypeName
			return
		}

		schema.Example = nil
		schema.Ref = fmt.Sprintf("#/components/schemas/%s", vType.String())

		if nextType := doc.searchType(pkgPath, vType.TypeName); nextType != nil {
			if doc.knownCount(vType.TypeName) < 2 {
				doc.knownInc(vType.TypeName)
				doc.schemas[vType.TypeName] = doc.walkVariable(typeName, pkgPath, nextType, varTags)
			}
		}

	case types.TMap:

		schema.Type = "object"
		schema.AdditionalProperties = doc.walkVariable(typeName, pkgPath, vType.Value, nil)

	case types.TArray:

		schema.Type = "array"
		schema.Maximum = vType.ArrayLen
		schema.Nullable = vType.IsSlice
		itemSchema := doc.walkVariable(typeName, pkgPath, vType.Next, nil)
		schema.Items = &itemSchema

	case types.Struct:

		schema.Type = "object"
		schema.Properties = make(swProperties)

		for _, field := range vType.Fields {
			if fieldName := jsonName(field); fieldName != "-" {
				schema.Properties[fieldName] = doc.walkVariable(field.Name, pkgPath, field.Type, tags.ParseTags(field.Docs))
			}
		}

	case types.TImport:

		schema.Example = nil
		schema.Ref = fmt.Sprintf("#/components/schemas/%s", vType.Next)

		if nextType := doc.searchType(vType.Import.Package, vType.Next.String()); nextType != nil {
			if doc.knownCount(vType.Next.String()) < 2 {
				doc.knownInc(vType.Next.String())
				doc.schemas[vType.Next.String()] = doc.walkVariable(typeName, vType.Import.Package, nextType, varTags)
			}
		}

	case types.TEllipsis:

		schema.Type = "array"
		itemSchema := doc.walkVariable(typeName, pkgPath, vType.Next, varTags)
		schema.Items = &itemSchema

	case types.TPointer:

		return doc.walkVariable(typeName, pkgPath, vType.Next, nil)

	case types.TInterface:

		schema.Type = "object"
		schema.Nullable = true

	default:
		doc.log.WithField("type", vType).Error("unknown type")
		return
	}
	return
}

func (doc *swagger) searchType(pkg, name string) (retType types.Type) {

	if retType = doc.parseType(pkg, name); retType == nil {

		pkgPath := mod.PkgModPath(pkg)

		if retType = doc.parseType(pkgPath, name); retType == nil {

			pkgPath = path.Join("./vendor", pkg)

			if retType = doc.parseType(pkgPath, name); retType == nil {

				pkgPath = doc.trimLocalPkg(pkg)
				retType = doc.parseType(pkgPath, name)
			}
		}
	}
	return
}

func (doc *swagger) parseType(relPath, name string) (retType types.Type) {

	pkgPath, _ := filepath.Abs(relPath)

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
		if srcFile, err = astra.ParseFile(filePath, astra.IgnoreConstants, astra.IgnoreMethods); err != nil {
			retErr = errors.Wrap(err, fmt.Sprintf("%s,%s", relPath, name))
			doc.log.WithError(err).Errorf("parse file %s", filePath)
			return err
		}

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

func (doc *swagger) trimLocalPkg(pkg string) (pgkPath string) {

	module := doc.getModName()

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

func (doc *swagger) getModName() (module string) {

	modFile, err := os.OpenFile("go.mod", os.O_RDONLY, os.ModePerm)

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

func castType(originName string) (typeName, format string) {

	typeName = originName

	switch originName {

	case "bool":
		typeName = "boolean"

	case "Interface":
		typeName = "object"

	case "time.Time":
		format = "date-time"
		typeName = "string"

	case "byte":
		format = "uint8"
		typeName = "number"

	case "[]byte":
		format = "byte"
		typeName = "string"

	case "float32", "float64":
		format = "float"
		typeName = "number"

	case "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		typeName = "number"
		format = originName
	}
	if strings.HasSuffix(originName, "UUID") {
		format = "uuid"
		typeName = "string"
	}
	return
}

func jsonName(fieldInfo types.StructField) (value string) {

	if fieldInfo.Variable.Name == "" {
		fieldInfo.Variable.Name = fieldInfo.Type.String()
	}
	if fieldInfo.Variable.Name[:1] != strings.ToUpper(fieldInfo.Variable.Name[:1]) {
		return "-"
	}
	value = fieldInfo.Name
	if tagValues, _ := fieldInfo.Tags["json"]; len(tagValues) > 0 {
		value = tagValues[0]
	}
	return
}

func (doc *swagger) knownCount(typeName string) int {
	if _, found := doc.knownTypes[typeName]; !found {
		return 0
	}
	return doc.knownTypes[typeName]
}

func (doc *swagger) knownInc(typeName string) {
	if _, found := doc.knownTypes[typeName]; !found {
		doc.knownTypes[typeName] = 0
	}
	doc.knownTypes[typeName]++
}
