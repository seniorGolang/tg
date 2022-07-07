package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/vetcher/go-astra/types"

	"github.com/seniorGolang/tg/v2/pkg/tags"
)

type clientTS struct {
	*Transport
	knownTypes map[string]int
	typeDefTs  map[string]typeDefTs
}

func (tr Transport) RenderClientTS(outDir string) (err error) {
	return newClientTS(&tr).render(outDir)
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
	outFilename := path.Join(outDir, fmt.Sprintf("%s-jsonrpc.ts", svc.lccName()))
	_ = os.Remove(outFilename)
	if err = os.MkdirAll(outDir, 0777); err != nil {
		return
	}
	var jsFile bytesWriter
	jsFile.add("export namespace %sApiTypes {\n", svc.Name)
	for _, method := range svc.methods {
		jsFile.add("export interface %sParams {\n", method.Name)
		for _, arg := range method.arguments() {
			switch vType := arg.Variable.Type.(type) {
			case types.TEllipsis:
				jsFile.add("%s?: %s[]\n", arg.Name, ts.walkVariable(arg.Name, svc.pkgPath, vType, method.tags).typeLink())
			default:
				jsFile.add("%s: %s\n", arg.Name, ts.walkVariable(arg.Name, svc.pkgPath, vType, method.tags).typeLink())
			}
		}
		jsFile.add("}\n")
		// if len(method.results()) > 0 {
		// 	var fields []string
		// 	jsFile.add("export interface %sResult {\n", method.Name)
		// 	for _, ret := range method.results() {
		// 		fields = append(fields, fmt.Sprintf("%s: %s\n", ret.Name, ts.walkVariable(ret.Name, svc.pkgPath, ret.Type, method.tags).typeLink()))
		// 	}
		// 	jsFile.add("}\n")
		// }
	}
	for _, def := range ts.typeDefTs {
		jsFile.add(def.ts())
	}
	jsFile.add("}\n\n")
	return ioutil.WriteFile(outFilename, jsFile.Bytes(), 0600)
}

type typeDefTs struct {
	name       string
	kind       string
	typeName   string
	nullable   bool
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

	var nullable string
	if def.nullable {
		nullable = "?"
	}
	js += "export interface " + def.name + " {\n"
	switch def.kind {
	// case "map":
	// 	js += fmt.Sprintf("Record<%s, %s>", castTypeTs(def.properties["key"].typeLink()), castTypeTs(def.properties["value"].typeLink()))
	// 	js += "}\n"
	// case "array":
	// 	js += fmt.Sprintf("%s: %s \n }\n", def.name, def.def())
	case "struct":
		for name, property := range def.properties {
			var pNullable string
			if property.nullable {
				pNullable = "?"
			}
			js += fmt.Sprintf("%s%s: %s\n", name, pNullable, castTypeTs(property.def()))
		}
		js += "}\n"
	default:
		js += fmt.Sprintf("%s%s: %s\n", def.name, nullable, castTypeTs(def.def()))
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
		if nextType := searchType(pkgPath, vType.TypeName); nextType != nil {
			if ts.knownCount(vType.TypeName) < 3 {
				ts.knownInc(vType.TypeName)
				ts.typeDefTs[vType.TypeName] = ts.walkVariable(typeName, pkgPath, nextType, varTags)
			}
			return ts.typeDefTs[vType.TypeName]
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
				for eField, def := range ts.typeDefTs[field.Type.String()].properties {
					schema.properties[eField] = def
				}
			}
		}
	case types.TImport:
		if nextType := searchType(vType.Import.Package, vType.Next.String()); nextType != nil {
			if ts.knownCount(vType.Next.String()) < 3 {
				ts.knownInc(vType.Next.String())
				ts.typeDefTs[vType.Next.String()] = ts.walkVariable(typeName, vType.Import.Package, nextType, varTags)
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
	case "time.Time":
		typeName = "string"
	case "[]byte":
		typeName = "string"
	case "float32", "float64":
		typeName = "number"
	case "byte", "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "time.Duration":
		typeName = "number"
	}
	if strings.HasSuffix(originName, "UUID") {
		typeName = "string"
	}
	if strings.HasSuffix(originName, "Decimal") {
		typeName = "number"
	}
	return
}
