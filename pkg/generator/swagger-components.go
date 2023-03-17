// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
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
	"unicode"

	"github.com/seniorGolang/tg/v2/pkg/astra"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	"github.com/seniorGolang/tg/v2/pkg/mod"
	"github.com/seniorGolang/tg/v2/pkg/tags"
	"github.com/seniorGolang/tg/v2/pkg/utils"
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

func (doc *swagger) registerComponents(typeName, pkgPath string, varType types.Type) { // nolint

	if doc.schemas == nil {
		doc.schemas = make(swSchemas)
	}
	doc.schemas[typeName] = doc.walkVariable(typeName, pkgPath, varType, nil)
}

func (doc *swagger) walkVariable(typeName, pkgPath string, varType types.Type, varTags tags.DocTags) (schema swSchema) {

	var found bool
	schemaName := doc.toSchemaName(typeName, pkgPath)
	if _, found = doc.schemas[schemaName]; found {
		schema.Ref = fmt.Sprintf("#/components/schemas/%s", schemaName)
		return
	}
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
		if enums := varTags.Value(tagEnums); enums != "" {
			schema.Enum = strings.Split(enums, ",")
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
	case types.TMap:
		schema.Type = "object"
		schema.AdditionalProperties = doc.walkVariable(typeName, pkgPath, vType.Value, nil)
	case types.TArray:
		schema.Type = "array"
		schema.Maximum = vType.ArrayLen
		schema.Nullable = vType.IsSlice
		itemSchema := doc.walkVariable(vType.Next.String(), pkgPath, vType.Next, nil)
		schema.Items = &itemSchema
	case types.Struct:
		schema.Type = "object"
		schema.Properties = make(swProperties)
		var inlined []swSchema
		for _, field := range vType.Fields {
			if fieldName, inline := jsonName(field); fieldName != "-" {
				embed := doc.walkVariable(field.Type.String(), pkgPath, field.Type, tags.ParseTags(field.Docs))
				if !inline {
					schema.Properties[fieldName] = embed
					continue
				}
				inlined = append(inlined, swSchema{Ref: embed.Ref})
			}
		}
		var allOf swSchema
		if len(inlined) != 0 {
			allOf.AllOf = append(allOf.AllOf, append(inlined, schema)...)
			schema = allOf
		}
	case types.TName:
		if types.IsBuiltin(varType) {
			schema.Type = vType.TypeName
			return
		}
		if nextType := searchType(pkgPath, vType.TypeName); nextType != nil {
			if doc.knownCount(schemaName) < 1 {
				doc.knownInc(schemaName)
				doc.schemas[schemaName] = doc.walkVariable(vType.TypeName, pkgPath, nextType, varTags)
			}
			schemaName = doc.toSchemaName(vType.TypeName, pkgPath)
			schema.Ref = fmt.Sprintf("#/components/schemas/%s", schemaName)
		}
	case types.TImport:
		if nextType := searchType(vType.Import.Package, vType.Next.String()); nextType != nil {
			schemaName = doc.toSchemaName(vType.Next.String(), vType.Import.Package)
			schema.Ref = fmt.Sprintf("#/components/schemas/%s", schemaName)
			if _, found = doc.schemas[schemaName]; found {
				return
			}
			doc.schemas[schemaName] = doc.walkVariable(nextType.String(), vType.Import.Package, nextType, varTags)
		}
	case types.TEllipsis:
		schema.Type = "array"
		itemSchema := doc.walkVariable(vType.Next.String(), pkgPath, vType.Next, varTags)
		schema.Items = &itemSchema
	case types.TPointer:
		if _, found = doc.schemas[schemaName]; found {
			return doc.schemas[schemaName]
		}
		schema = doc.walkVariable(vType.Next.String(), pkgPath, vType.Next, varTags)
	case types.TInterface:
		schema.Type = "object"
		schema.Nullable = true
	default:
		doc.log.WithField("type", vType).Error("unknown type")
	}
	return
}

func (doc *swagger) toSchemaName(typeName, pkgPath string) string {

	schemaName := strings.TrimPrefix(typeName, "*")
	if !strings.Contains(schemaName, ".") {
		schemaName = fmt.Sprintf("%s.%s", filepath.Base(pkgPath), schemaName)
	}
	return schemaName
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
			// doc.log.WithError(err).Errorf("parse file %s", filePath)
			return nil
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
	case "Interface", "json.RawMessage":
		typeName = "object"
	case "time.Time":
		format = "date-time"
		typeName = "string"
	case "sql.NullTime":
		format = "date-time"
		typeName = "string"
	case "byte":
		format = "uint8"
		typeName = "number"
	case "[]byte":
		format = "byte"
		typeName = "string"
	case "fiber.Cookie":
		typeName = "string"
	case "snowflake.ID":
		typeName = "string"
	case "JSON":
		format = "byte"
		typeName = "string"
	case "float32", "float64":
		format = "float"
		typeName = "number"
	case "time.Duration":
		typeName = "number"
		format = "int64"
	case "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		typeName = "number"
		format = originName
	}
	if !strings.Contains(originName, "[") && strings.HasSuffix(originName, "Decimal") {
		typeName = "number"
	}
	if !strings.Contains(originName, "[") && strings.HasSuffix(originName, "UUID") {
		format = "uuid"
		typeName = "string"
	}
	return
}

func jsonName(fieldInfo types.StructField) (value string, inline bool) {

	if fieldInfo.Variable.Name == "" {
		fieldInfo.Variable.Name = fieldInfo.Type.String()
	}
	// if fieldInfo.Variable.Name[:1] != strings.ToUpper(fieldInfo.Variable.Name[:1]) {
	// 	return "-", false
	// }
	value = fieldInfo.Name
	if tagValues, _ := fieldInfo.Tags["json"]; len(tagValues) > 0 { // nolint
		value = tagValues[0]
		if len(tagValues) == 2 {
			inline = tagValues[1] == "inline"
		}
	}
	if isLowerStart(fieldInfo.Variable.Name) {
		value = "-"
	}
	return
}

func isLowerStart(s string) bool {

	for _, r := range s {
		if unicode.IsLower(r) && unicode.IsLetter(r) {
			return true
		}
		break
	}
	return false
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

func (doc *swagger) aliasCount(typeName string) int { // nolint
	if _, found := doc.aliasTypes[typeName]; !found {
		return 0
	}
	return doc.knownTypes[typeName]
}

func (doc *swagger) aliasInc(typeName string) { // nolint
	if _, found := doc.aliasTypes[typeName]; !found {
		doc.knownTypes[typeName] = 0
	}
	doc.knownTypes[typeName]++
}
