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

func (doc *swagger) registerStruct(name, pkgPath string, mTags tags.DocTags, fields []types.StructField) (structType types.Struct) {

	if len(fields) == 0 {
		doc.schemas[name] = swSchema{Type: "object"}
		return
	}
	if doc.schemas == nil {
		doc.schemas = make(swSchemas)
	}
	structType = types.Struct{
		Base: types.Base{Name: name, Docs: mTags.ToDocs()},
	}
	var required []string
	for _, field := range fields {
		field.Docs = mTags.Sub(utils.ToLowerCamel(field.Name)).ToDocs()
		structType.Fields = append(structType.Fields, field)
		if fieldName, inline := jsonName(field); !inline {
			required = append(required, fieldName)
		}
	}
	schema := doc.walkVariable(name, pkgPath, structType, mTags)
	schema.Required = required
	doc.schemas[name] = schema
	return
}

func (doc *swagger) registerComponents(typeName, pkgPath string, varType types.Type) { // nolint

	if doc.schemas == nil {
		doc.schemas = make(swSchemas)
	}
	doc.schemas[typeName] = doc.walkVariable(typeName, pkgPath, varType, nil)
}

func (doc *swagger) walkVariable(typeName, pkgPath string, varType types.Type, varTags tags.DocTags) (schema swSchema) {

	var found bool
	typeName = doc.normalizeTypeName(typeName, pkgPath)
	if _, found = doc.schemas[typeName]; found {
		return doc.toSchema(typeName)
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
		return
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
					if tags.ParseTags(field.Docs).IsSet(tagRequired) {
						schema.Required = append(schema.Required, fieldName)
					}
					continue
				}
				if len(embed.AllOf) != 0 {
					inlined = append(inlined, embed.AllOf...)
				} else {
					inlined = append(inlined, swSchema{Ref: embed.Ref})
				}
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
		if schema, found = doc.schemas[typeName]; found {
			return
		}
		if doc.knownCount[typeName] > 0 {
			return doc.toSchema(doc.normalizeTypeName(vType.TypeName, pkgPath))
		}
		if nextType := searchType(pkgPath, vType.TypeName); nextType != nil {

			doc.knownCount[typeName]++
			typeName = doc.normalizeTypeName(vType.String(), pkgPath)
			if _, found = doc.schemas[typeName]; !found {
				doc.schemas[typeName] = doc.walkVariable(vType.TypeName, pkgPath, nextType, varTags)
			}
			return doc.toSchema(doc.normalizeTypeName(vType.String(), pkgPath))
		}

	case types.TImport:
		if nextType := searchType(vType.Import.Package, vType.Next.String()); nextType != nil {
			if _, found = doc.schemas[typeName]; !found {
				doc.schemas[doc.normalizeTypeName(vType.Next.String(), vType.Import.Package)] = doc.walkVariable(nextType.String(), vType.Import.Package, nextType, varTags)
			}
			return doc.toSchema(doc.normalizeTypeName(vType.Next.String(), vType.Import.Package))
		}
	case types.TEllipsis:
		schema.Type = "array"
		itemSchema := doc.walkVariable(vType.Next.String(), pkgPath, vType.Next, varTags)
		schema.Items = &itemSchema
	case types.TPointer:
		if _, found = doc.schemas[typeName]; found {
			schema.OneOf = append(schema.OneOf, doc.schemas[typeName], swSchema{Nullable: true})
			return
		}
		schema.OneOf = append(schema.OneOf, doc.walkVariable(vType.Next.String(), pkgPath, vType.Next, varTags), swSchema{Nullable: true})
	case types.TInterface:
		schema.Type = "object"
		schema.Nullable = true
	default:
		doc.log.WithField("type", vType).Error("unknown type")
	}
	return
}

func (doc *swagger) normalizeTypeName(typeName string, pkgPath string) string {

	typeName = strings.TrimPrefix(typeName, "*")
	if !strings.Contains(typeName, ".") {
		typeName = fmt.Sprintf("%s.%s", filepath.Base(pkgPath), typeName)
	}
	return typeName
}

func (doc *swagger) toSchema(typeName string) (schema swSchema) {

	isPointer := strings.HasPrefix(typeName, "*")
	if isPointer {
		schema.OneOf = append(schema.OneOf, swSchema{Ref: fmt.Sprintf("#/components/schemas/%s", typeName)}, swSchema{Nullable: true})
		return
	}
	schema.Ref = fmt.Sprintf("#/components/schemas/%s", typeName)
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
	case "any", "Interface", "json.RawMessage":
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
		typeName = "string"
	}
	if !strings.Contains(originName, "[") && strings.HasSuffix(originName, "UUID") {
		format = "uuid"
		typeName = "string"
	}
	return
}

func jsonName(fieldInfo types.StructField) (value string, inline bool) {

	if fieldInfo.Name == "" {
		fieldInfo.Name = fieldInfo.Type.String()
	}
	value = fieldInfo.Name
	if tagValues, _ := fieldInfo.Tags["json"]; len(tagValues) > 0 { // nolint
		value = tagValues[0]
		if len(tagValues) == 2 {
			inline = tagValues[1] == "inline"
		}
	}
	if isLowerStart(fieldInfo.Name) && !inline {
		value = "-"
	}
	return
}

func isLowerStart(s string) bool {

	for _, r := range s {
		if unicode.IsLower(r) && unicode.IsLetter(r) {
			return true
		}
		break // nolint
	}
	return false
}
