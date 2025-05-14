// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-jsonrpc.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	"github.com/seniorGolang/tg/v2/pkg/tags"
)

type clientTS struct {
	*Transport
	knownTypes map[string]int
	typeDefTs  map[string]typeDefTs
}

func (tr *Transport) RenderClientTS(outDir string) (err error) {
	return newClientTS(tr).render(outDir)
}

func newClientTS(tr *Transport) (js *clientTS) {
	js = &clientTS{
		Transport:  tr,
		knownTypes: make(map[string]int),
		typeDefTs:  make(map[string]typeDefTs),
	}
	return
}

func (ts *clientTS) render(outDir string) (err error) {

	if err = tsCopyTo("jsonrpc", outDir); err != nil {
		return err
	}
	for _, name := range ts.serviceKeys() {
		svc := ts.services[name]
		if !svc.isJsonRPC() {
			continue
		}
		if err = ts.renderService(svc, outDir); err != nil {
			return err
		}
	}
	return
}

func (ts *clientTS) renderService(svc *service, outDir string) (err error) {

	ts.knownTypes = make(map[string]int)
	ts.typeDefTs = make(map[string]typeDefTs)
	outFilename := path.Join(outDir, fmt.Sprintf("%s.ts", svc.lccName()))
	_ = os.Remove(outFilename)
	if err = os.MkdirAll(outDir, 0777); err != nil {
		return
	}
	var jsFile bytesWriter
	jsFile.add("import {rpcClient} from \"./jsonrpc/jsonrpc\";\n\n")
	jsFile.add("export namespace %sAPI {\n\n", svc.Name)
	jsFile.add(`export const RPC = (headers?: Record<string, string>) => {
        return rpcClient<Methods>({
            url: "%s",
            getHeaders: () => headers
        })
    }
`, svc.batchPath())
	jsFile.add("export type Methods = {\n")
	for _, method := range svc.methods {
		jsFile.add("%s(params: {%s}) : {%s}\n",
			method.Name,
			ts.paramsToFuncParams(svc.pkgPath, method.tags, method.argsWithoutContext()),
			ts.paramsToFuncParams(svc.pkgPath, method.tags, method.resultsWithoutError()),
		)
	}
	jsFile.add("}\n")
	for _, def := range ts.typeDefTs {
		jsFile.add(def.ts()) // nolint
	}
	jsFile.add("}\n\n")
	return os.WriteFile(outFilename, jsFile.Bytes(), 0600)
}

func (ts *clientTS) paramsToFuncParams(pkgPath string, tags tags.DocTags, vars []types.Variable) string {

	var params = make([]string, 0, len(vars))
	for _, arg := range vars {
		params = append(params, fmt.Sprintf("%s: %s", arg.Name, ts.walkVariable(arg.Name, pkgPath, arg.Type, tags).typeLink()))
	}
	return strings.Join(params, ",")
}

type typeDefTs struct {
	name       string
	kind       string
	typeName   string
	nullable   bool
	value      interface{}
	properties map[string]typeDefTs
}

func (def typeDefTs) def() (prop string) {
	switch def.kind {
	case "map":
		key := def.properties["key"]
		value := def.properties["value"]
		return fmt.Sprintf("Record<%s, %s>", castTypeTs(key.typeLink()), castTypeTs(value.typeLink()))
	case "array":
		item := def.properties["item"]
		return fmt.Sprintf("%s[]", item.typeLink())
	case "struct":
		return def.name
	case "scalar":
		return def.typeName
	default:
		return castTypeTs(def.kind)
	}
}

func (def typeDefTs) ts() (js string) {

	switch def.kind {
	case "constant":
		if len(def.properties) > 1 {
			if def.typeName == "iota" {
				var cnt int
				for key := range def.properties {
					js += fmt.Sprintf("export const %s = %d;\n", key, cnt)
					cnt++
				}
			} else {
				js += "export enum " + def.typeName + " {\n"
				for key := range def.properties {
					js += fmt.Sprintf("%s,\n", key)
				}
				js += "}\n"
			}
		} else {
			js += fmt.Sprintf("export const %s = %v;\n", def.name, def.value)
		}
	case "struct":
		js += "export interface " + def.name + " {\n"
		for name, property := range def.properties {
			var pNullable string
			if property.nullable {
				pNullable = "?"
			}
			js += fmt.Sprintf("%s%s: %s\n", name, pNullable, castTypeTs(property.def()))
		}
		js += "}\n"
	}
	return
}

func (def typeDefTs) typeLink() (link string) {

	switch def.kind {
	case "map":
		return fmt.Sprintf("Record<%s, %s>", castTypeTs(def.properties["key"].typeLink()), castTypeTs(def.properties["value"].typeLink()))
	case "array":
		return fmt.Sprintf("%s[]", castTypeTs(def.properties["item"].typeLink()))
	case "scalar":
		return def.typeName
	default:
		return castTypeTs(def.name)
	}
}

func (ts *clientTS) walkVariable(typeName, pkgPath string, varType types.Type, varTags tags.DocTags) (schema typeDefTs) {

	schema.name = typeName
	schema.typeName = varType.String()
	schema.properties = make(map[string]typeDefTs)
	if fl, ok := varTags["nullable"]; ok {
		schema.nullable = fl == "true"
	}
	if newType := castTypeTs(varType.String()); newType != varType.String() {
		schema.kind = "scalar"
		schema.typeName = newType
		return
	}
	switch vType := varType.(type) {
	case types.TName:
		schema.kind = vType.TypeName
		if types.IsBuiltin(varType) {
			schema.name = typeName
			schema.kind = "scalar"
			schema.typeName = vType.String()
			return
		}
		if nextType, _ := ts.searchType(pkgPath, vType.TypeName); nextType != nil {
			if ts.knownCount(vType.TypeName) < 3 {
				ts.knownInc(vType.TypeName)
				ts.typeDefTs[vType.TypeName] = ts.walkVariable(typeName, pkgPath, nextType, varTags)
			}
			return typeDefTs{
				kind:     "scalar",
				name:     vType.TypeName,
				typeName: vType.TypeName,
			}
		}
	case types.TMap:
		schema.kind = "map"
		schema.typeName = "map"
		schema.nullable = true
		key := ts.walkVariable(typeName, pkgPath, vType.Key, nil)
		value := ts.walkVariable(typeName, pkgPath, vType.Value, nil)
		if !types.IsBuiltin(vType.Key) {
			ts.typeDefTs[vType.Key.String()] = key
		}
		if !types.IsBuiltin(vType.Value) {
			switch vType.Value.(type) {
			case types.TInterface:
			default:
				ts.typeDefTs[vType.Value.String()] = value
			}
		}
		schema.properties["key"] = key
		schema.properties["value"] = value
	case types.TArray:
		schema.kind = "array"
		schema.typeName = "array"
		schema.nullable = true
		schema.properties["item"] = ts.walkVariable(vType.Next.String(), pkgPath, vType.Next, nil)
	case types.Struct:
		schema.name = vType.Name
		schema.kind = "struct"
		schema.typeName = "struct"
		for _, field := range vType.Fields {
			if fieldName, inline := jsonName(field); fieldName != "-" {
				embed := ts.walkVariable(field.Name, pkgPath, field.Type, tags.ParseTags(field.Docs))
				if !inline {
					schema.properties[fieldName] = embed
					continue
				}
				inlineTokens := strings.Split(field.Type.String(), ".")
				embed = ts.typeDefTs[inlineTokens[len(inlineTokens)-1]]
				for eField, def := range embed.properties {
					schema.properties[eField] = def
				}
			}
		}
	case types.TImport:
		if nextType, constants := ts.searchType(vType.Import.Package, vType.Next.String()); nextType != nil {
			if ts.knownCount(vType.Next.String()) < 10 {
				ts.knownInc(vType.Next.String())
				ts.typeDefTs[vType.Next.String()] = ts.walkVariable(typeName, vType.Import.Package, nextType, varTags)
			}
			for _, c := range constants {
				def := typeDefTs{
					name:     c.Name,
					kind:     "constant",
					value:    c.Value,
					typeName: c.Type.String(),
				}
				def.properties = make(map[string]typeDefTs)
				for _, v := range c.Constants {
					def.properties[v.Name] = typeDefTs{
						kind:     "constant",
						name:     v.Name,
						value:    v.Value,
						typeName: v.Type.String(),
					}
				}
				ts.typeDefTs[c.Type.String()] = def
			}
			return ts.typeDefTs[vType.Next.String()]
		}
	case types.TEllipsis:
		schema.kind = "array"
		schema.typeName = "array"
		schema.nullable = true
		schema.properties[vType.String()] = ts.walkVariable(typeName, pkgPath, vType.Next, varTags)
		if !types.IsBuiltin(vType.Next) {
			ts.typeDefTs[vType.Next.String()] = ts.walkVariable(typeName, pkgPath, vType.Next, varTags)
		}
	case types.TPointer:
		return ts.walkVariable(typeName, pkgPath, vType.Next, tags.DocTags{"nullable": "true"})
	case types.TInterface:
		schema.kind = "scalar"
		schema.name = "interface"
		schema.typeName = "interface"
		schema.nullable = true
	}
	return
}

func (ts *clientTS) knownCount(typeName string) int {
	if _, found := ts.knownTypes[typeName]; !found {
		return 0
	}
	return ts.knownTypes[typeName]
}

func (ts *clientTS) knownInc(typeName string) {
	if _, found := ts.knownTypes[typeName]; !found {
		ts.knownTypes[typeName] = 0
	}
	ts.knownTypes[typeName]++
}

func castTypeTs(originName string) (typeName string) {

	typeName = originName
	switch originName {
	case "JSON":
		typeName = "any"
	case "bool":
		typeName = "boolean"
	case "interface":
		typeName = "any"
	case "gorm.DeletedAt":
		typeName = "Date"
	case "time.Time":
		typeName = "Date"
	case "[]byte":
		typeName = "string"
	case "float32", "float64":
		typeName = "number"
	case "byte", "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "time.Duration":
		typeName = "number"
	}
	if strings.HasSuffix(originName, "NullTime") {
		typeName = "Date"
	}
	if strings.HasSuffix(originName, "RawMessage") {
		typeName = "any"
	}
	if strings.HasSuffix(originName, "UUID") {
		typeName = "string"
	}
	if strings.HasSuffix(originName, "Decimal") {
		typeName = "number"
	}
	return
}
