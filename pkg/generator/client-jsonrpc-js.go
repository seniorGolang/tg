// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-jsonrpc.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
	"github.com/seniorGolang/tg/v2/pkg/tags"
	"github.com/seniorGolang/tg/v2/pkg/utils"
)

type clientJS struct {
	*Transport
	knownTypes map[string]int
	typeDef    map[string]typeDefJs
}

func (tr *Transport) RenderClientJS(outDir string) (err error) {
	return newClientJS(tr).render(outDir)
}

func newClientJS(tr *Transport) (js *clientJS) {
	js = &clientJS{
		Transport:  tr,
		knownTypes: make(map[string]int),
		typeDef:    make(map[string]typeDefJs),
	}
	return
}

func (tr *Transport) RenderPackageNPM(outJs, outPkg string) (err error) {

	type npmPackage struct {
		Name          string   `json:"name"`
		Version       string   `json:"version"`
		Description   string   `json:"description"`
		Main          string   `json:"main"`
		Files         []string `json:"files"`
		Private       bool     `json:"private"`
		PublishConfig struct {
			Registry string `json:"registry"`
		} `json:"publishConfig"`
		Scripts struct {
			Test string `json:"test"`
		} `json:"scripts"`
		Author  string `json:"author"`
		License string `json:"license"`
	}
	modPath, _ := filepath.Abs(tr.modPath)
	jsPath, _ := filepath.Abs(path.Join(outJs, "jsonrpc-client.js"))
	jsPath = strings.TrimPrefix(jsPath, strings.TrimSuffix(modPath, "/go.mod"))
	data := npmPackage{
		Name:        tr.tags.Value(tagNameNPM),
		Version:     tr.tags.Value(tagAppVersion, "v1.0.0"),
		Description: tr.tags.Value(tagDesc),
		Main:        jsPath,
		Files:       []string{jsPath},
		Private:     tr.tags.ValueBool(tagPrivateNPM),
		PublishConfig: struct {
			Registry string `json:"registry"`
		}{
			Registry: tr.tags.Value(tagRegistryNPM),
		},
		Scripts: struct {
			Test string `json:"test"`
		}{
			Test: "echo 'No tests to run'",
		},
		Author:  tr.tags.Value(tagAuthor),
		License: tr.tags.Value(tagLicense),
	}
	var bytes []byte
	if bytes, err = json.MarshalIndent(data, "", "    "); err != nil {
		return
	}
	return os.WriteFile(path.Join(outPkg, "package.json"), bytes, 0600)
}

func (js *clientJS) render(outDir string) (err error) {

	outFilename := path.Join(outDir, "jsonrpc-client.js")
	_ = os.Remove(outFilename)
	if err = os.MkdirAll(outDir, 0777); err != nil {
		return
	}
	var jsFile bytesWriter
	jsFile.add(jsonRPCClientBase)
	for _, name := range js.serviceKeys() {
		svc := js.services[name]
		if !svc.isJsonRPC() {
			continue
		}
		jsFile.add("class JSONRPCClient%s {\n", svc.Name)
		jsFile.add("constructor(transport) {\n")
		jsFile.add("this.scheduler = new JSONRPCScheduler(transport);\n")
		jsFile.add("}\n\n")
		for _, method := range svc.methods {
			jsFile.add("/**\n")
			if comment := method.tags.Value("summary", ""); comment != "" {
				jsFile.add("* %s\n", comment)
				jsFile.add("*\n")
			}
			for _, arg := range method.arguments() {
				switch vType := arg.Type.(type) {
				case types.TEllipsis:
					jsFile.add("* @param {...%s} %s\n", js.walkVariable(arg.Name, svc.pkgPath, vType, method.tags).typeLink(), arg.Name)
				default:
					jsFile.add("* @param {%s} %s\n", js.walkVariable(arg.Name, svc.pkgPath, vType, method.tags).typeLink(), arg.Name)
				}
			}
			if len(method.results()) > 0 {
				var fields []string
				jsFile.add("* @return {PromiseLike<{")
				for _, ret := range method.results() {
					fields = append(fields, fmt.Sprintf("%s: %s", ret.Name, js.walkVariable(ret.Name, svc.pkgPath, ret.Type, method.tags).typeLink()))
				}
				jsFile.add(strings.Join(fields, ",")) // nolint
				jsFile.add("}>}\n")
			}
			jsFile.add("**/\n")
			jsFile.add("%s(", method.lccName())
			var fields []string
			for _, arg := range method.arguments() {
				var prefix string
				if _, ok := arg.Type.(types.TEllipsis); ok {
					prefix = "..."
				}
				fields = append(fields, prefix+utils.ToLowerCamel(arg.Name))
			}
			jsFile.add(strings.Join(fields, ",")) // nolint
			jsFile.add(") {\n")
			jsFile.add("return this.scheduler.__scheduleRequest(\"%s\", {", svc.lccName()+"."+method.lccName())
			fields = []string{}
			for _, arg := range method.arguments() {
				fields = append(fields, fmt.Sprintf("%[1]s:%[1]s", utils.ToLowerCamel(arg.Name)))
			}
			jsFile.add(strings.Join(fields, ",")) // nolint
			jsFile.add("}).catch(e => { throw ")
			jsFile.add("%sConvertError(e)", utils.ToLowerCamel(method.fullName()))
			jsFile.add("; })\n")
			jsFile.add("}\n")
		}
		jsFile.add("}\n\n")
	}
	jsFile.add("class JSONRPCClient {\n")
	jsFile.add("constructor(transport) {\n")
	for _, name := range js.serviceKeys() {
		svc := js.services[name]
		if !svc.isJsonRPC() {
			continue
		}
		jsFile.add("this.%s = new JSONRPCClient%s(transport);\n", svc.lccName(), svc.Name)
	}
	jsFile.add("}\n")
	jsFile.add("}\n")
	jsFile.add("export default JSONRPCClient\n\n")
	for _, name := range js.serviceKeys() {
		svc := js.services[name]
		if !svc.isJsonRPC() {
			continue
		}
		for _, method := range svc.methods {
			jsFile.add("function %sConvertError(e) {\n", utils.ToLowerCamel(method.fullName()))
			jsFile.add("switch(e.code) {\n")
			jsFile.add("default:\n")
			jsFile.add("return new JSONRPCError(e.message, \"UnknownError\", e.code, e.data);\n")
			jsFile.add("}\n}\n")
		}
	}
	for _, def := range js.typeDef {
		jsFile.add(def.js()) // nolint
	}
	return os.WriteFile(outFilename, jsFile.Bytes(), 0600)
}

type typeDefJs struct {
	name       string
	kind       string
	typeName   string
	properties map[string]typeDefJs
}

func (def typeDefJs) def() (prop string) {
	switch def.kind {
	case "map":
		key := def.properties["key"]
		value := def.properties["value"]
		return fmt.Sprintf("Object<%s, %s>", key.typeLink(), value.typeLink())
	case "array":
		item := def.properties["item"]
		return fmt.Sprintf("Array<%s>", item.typeLink())
	case "struct":
		return fmt.Sprintf("Object<%s>", def.name)
	case "scalar":
		return def.typeName
	default:
		return castTypeJs(def.kind)
	}
}

func (def typeDefJs) js() (js string) {

	js += "/**\n"
	switch def.kind {
	case "map":
		js += fmt.Sprintf("* @typedef %s %s \n", def.def(), def.name)
	case "array":
		js += fmt.Sprintf("* @typedef %s %s \n", def.def(), def.name)
	case "struct":
		js += fmt.Sprintf("* @typedef {Object} %s\n", def.name)
		for name, property := range def.properties {
			js += fmt.Sprintf("* @property {%s} %s\n", property.def(), name)
		}
	default:
		js += fmt.Sprintf("* @typedef {%s} %s\n", def.def(), def.name)
	}
	js += "*/\n\n"
	return
}

func (def typeDefJs) typeLink() (link string) {
	switch def.kind {
	case "map":
		return fmt.Sprintf("Object<%s,%s>", castTypeJs(def.properties["key"].typeLink()), castTypeJs(def.properties["value"].typeLink()))
	case "array":
		return fmt.Sprintf("Array<%s>", castTypeJs(def.properties["item"].typeLink()))
	case "scalar":
		return def.typeName
	default:
		return castTypeJs(def.name)
	}
}

func (js *clientJS) walkVariable(typeName, pkgPath string, varType types.Type, varTags tags.DocTags) (schema typeDefJs) {

	schema.name = typeName
	schema.typeName = varType.String()
	schema.properties = make(map[string]typeDefJs)
	if newType := castTypeJs(varType.String()); newType != varType.String() {
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
		if nextType := searchType(pkgPath, vType.TypeName); nextType != nil {
			if js.knownCount(vType.TypeName) < 3 {
				js.knownInc(vType.TypeName)
				js.typeDef[vType.TypeName] = js.walkVariable(typeName, pkgPath, nextType, varTags)
			}
			return js.typeDef[vType.TypeName]
		}
	case types.TMap:
		schema.kind = "map"
		schema.typeName = "map"
		key := js.walkVariable(typeName, pkgPath, vType.Key, nil)
		value := js.walkVariable(typeName, pkgPath, vType.Value, nil)
		if !types.IsBuiltin(vType.Key) {
			js.typeDef[vType.Key.String()] = key
		}
		if !types.IsBuiltin(vType.Value) {
			switch vType.Value.(type) {
			case types.TInterface:
			default:
				js.typeDef[vType.Value.String()] = value
			}
		}
		schema.properties["key"] = key
		schema.properties["value"] = value
	case types.TArray:
		schema.kind = "array"
		schema.typeName = "array"
		schema.properties["item"] = js.walkVariable(vType.Next.String(), pkgPath, vType.Next, nil)
	case types.Struct:
		schema.name = vType.Name
		schema.kind = "struct"
		schema.typeName = "struct"
		for _, field := range vType.Fields {
			if fieldName, inline := jsonName(field); fieldName != "-" {
				embed := js.walkVariable(field.Name, pkgPath, field.Type, tags.ParseTags(field.Docs))
				if !inline {
					schema.properties[fieldName] = embed
					continue
				}
				for eField, def := range js.typeDef[field.Type.String()].properties {
					schema.properties[eField] = def
				}
			}
		}
	case types.TImport:
		if nextType := searchType(vType.Import.Package, vType.Next.String()); nextType != nil {
			if js.knownCount(vType.Next.String()) < 3 {
				js.knownInc(vType.Next.String())
				js.typeDef[vType.Next.String()] = js.walkVariable(typeName, vType.Import.Package, nextType, varTags)
			}
			return js.typeDef[vType.Next.String()]
		}
	case types.TEllipsis:
		schema.kind = "array"
		schema.typeName = "array"
		schema.properties[vType.String()] = js.walkVariable(typeName, pkgPath, vType.Next, varTags)
		if !types.IsBuiltin(vType.Next) {
			js.typeDef[vType.Next.String()] = js.walkVariable(typeName, pkgPath, vType.Next, varTags)
		}
	case types.TPointer:
		return js.walkVariable(typeName, pkgPath, vType.Next, nil)
	case types.TInterface:
		schema.kind = "scalar"
		schema.name = "interface"
		schema.typeName = "interface"
	}
	return
}

func (js *clientJS) knownCount(typeName string) int {
	if _, found := js.knownTypes[typeName]; !found {
		return 0
	}
	return js.knownTypes[typeName]
}

func (js *clientJS) knownInc(typeName string) {
	if _, found := js.knownTypes[typeName]; !found {
		js.knownTypes[typeName] = 0
	}
	js.knownTypes[typeName]++
}

func castTypeJs(originName string) (typeName string) {
	typeName = originName
	switch originName {
	case "bool":
		typeName = "boolean"
	case "interface":
		typeName = "Object"
	case "time.Time":
		typeName = "string"
	case "[]byte":
		typeName = "string"
	case "float32", "float64":
		typeName = "number"
	case "byte", "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "time.Duration":
		typeName = "number"
	}
	if strings.HasSuffix(originName, "RawMessage") {
		typeName = "string"
	}
	if strings.HasSuffix(originName, "UUID") {
		typeName = "string"
	}
	if strings.HasSuffix(originName, "Decimal") {
		typeName = "number"
	}
	return
}
