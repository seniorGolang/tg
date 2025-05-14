// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (method.go at 18.06.2020, 12:27) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/tags"
	"github.com/seniorGolang/tg/v2/pkg/utils"
)

type method struct {
	*types.Function

	log logrus.FieldLogger

	svc  *service
	tags tags.DocTags

	argFields    []types.StructField
	resultFields []types.StructField

	pathToArg   map[string]string
	paramToArg  map[string]string
	headerToVar map[string]string
	cookieToVar map[string]string
	cookieToArg map[string]string
	cookieToRet map[string]string
}

func newMethod(log logrus.FieldLogger, svc *service, fn *types.Function) (m *method) {

	m = &method{
		Function: fn,
		log:      log,
		svc:      svc,
		tags:     tags.ParseTags(fn.Docs).Merge(svc.tags),
	}
	m.argFields = m.varsToFields(m.argsWithoutContext(), m.tags, m.argCookieMap(), m.varHeaderMap())
	m.resultFields = m.varsToFields(m.resultsWithoutError(), m.tags, m.retCookieMap(), m.varHeaderMap())
	return
}

func (m *method) fullName() string {
	return fmt.Sprintf("%s.%s", utils.ToLowerCamel(m.svc.Name), utils.ToLowerCamel(m.Name))
}

func (m *method) lcName() string {
	return strings.ToLower(m.Name)
}

func (m *method) lccName() string {
	return utils.ToLowerCamel(m.Name)
}

func (m *method) requestStructName() string {
	return "request" + m.svc.Name + m.Name
}

func (m *method) responseStructName() string {
	return "response" + m.svc.Name + m.Name
}

func (m *method) httpPath(withoutPrefix ...bool) string {
	var elements []string
	if len(withoutPrefix) == 0 {
		elements = append(elements, "/")
	}
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName()))
	return path.Join(append(elements, prefix, urlPath)...)
}

func (m *method) httpPathSwagger(withoutPrefix ...bool) string {
	var elements []string
	if len(withoutPrefix) == 0 {
		elements = append(elements, "/")
	}
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName()))
	pathItems := strings.Split(urlPath, "/")
	var pathTokens []string // nolint:prealloc
	for _, pathItem := range pathItems {
		if strings.HasPrefix(pathItem, ":") {
			pathTokens = append(pathTokens, fmt.Sprintf("{%s}", strings.TrimPrefix(pathItem, ":")))
			continue
		}
		pathTokens = append(pathTokens, pathItem)
	}
	urlPath = strings.Join(pathTokens, "/")
	return path.Join(append(elements, prefix, urlPath)...)
}

func (m *method) jsonrpcPath(withoutPrefix ...bool) string {
	var elements []string
	if len(withoutPrefix) == 0 {
		elements = append(elements, "/")
	}
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := formatPathURL(m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName())))
	return path.Join(append(elements, prefix, urlPath)...)
}

func (m *method) httpMethod() string {

	switch strings.ToUpper(m.tags.Value(tagMethodHTTP)) {
	case "GET":
		return "get"
	case "PUT":
		return "put"
	case "PATCH":
		return "patch"
	case "DELETE":
		return "delete"
	case "OPTIONS":
		return "options"
	default:
		return "post"
	}
}

func formatPathURL(url string) string {
	return strings.Split(url, ":")[0]
}

func (m *method) isHTTP() bool {
	return m.svc.tags.Contains(tagServerHTTP) && m.tags.Contains(tagMethodHTTP)
}

func (m *method) isJsonRPC() bool {
	return m.svc.tags.Contains(tagServerJsonRPC) && !m.tags.Contains(tagMethodHTTP)
}

func (m *method) handlerQual() (pkgPath, handler string) {

	if !m.tags.Contains(tagHandler) {
		return
	}
	if tokens := strings.Split(m.tags.Value(tagHandler), ":"); len(tokens) == 2 {
		return tokens[0], tokens[1]
	}
	return
}

func (m *method) arguments() (vars []types.StructField) {

	argsAll := m.fieldsArgument()

	for _, arg := range argsAll {

		_, inPath := m.argPathMap()[arg.Name]
		_, inArgs := m.argParamMap()[arg.Name]
		_, inHeader := m.varHeaderMap()[arg.Name]
		_, inCookie := m.varCookieMap()[arg.Name]

		if !inArgs && !inPath && !inHeader && !inCookie {

			if jsonTags := arg.Tags["json"]; len(jsonTags) == 0 {
				if arg.Tags == nil {
					arg.Tags = map[string][]string{"json": {arg.Name}}
				} else {
					arg.Tags["json"] = []string{arg.Name}
				}
			}
			arg.Name = utils.ToCamel(arg.Name)
			vars = append(vars, arg)
		}
	}
	return
}

func (m *method) results() (vars []types.StructField) {

	argsAll := m.fieldsResult()

	for _, arg := range argsAll {

		_, inHeader := m.varHeaderMap()[arg.Name]
		_, inCookie := m.varCookieMap()[arg.Name]

		if !inHeader && !inCookie {
			arg.Tags = map[string][]string{"json": {arg.Name}}
			arg.Name = utils.ToCamel(arg.Name)
			vars = append(vars, arg)
		}
	}
	return
}

func (m *method) urlArgs(errStatement func(arg, header string) *Statement) (g *Statement) {

	return m.argFromString("urlParam", m.argPathMap(),
		func(srcName string) Code {
			return Id(_ctx_).Dot("Params").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (m *method) urlParams(errStatement func(arg, header string) *Statement) (block *Statement) {

	return m.argFromString("urlParam", m.argParamMap(),
		func(srcName string) Code {
			return Id(_ctx_).Dot("Query").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (m *method) httpArgHeaders(errStatement func(arg, header string) *Statement) (block *Statement) {

	return m.argFromString("header", m.varHeaderMap(),
		func(srcName string) Code {
			srcName = strings.TrimPrefix(srcName, "!")
			return String().Call(Id(_ctx_).Dot("Request").Call().Dot("Header").Dot("Peek").Call(Lit(srcName)))
		},
		errStatement,
	)
}

func (m *method) httpCookies(errStatement func(arg, header string) *Statement) (block *Statement) {

	return m.argFromString("cookie", m.varCookieMap(),
		func(srcName string) Code {
			srcName = strings.TrimPrefix(srcName, "!")
			return Id(_ctx_).Dot("Cookies").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (m *method) argFromString(typeName string, varMap map[string]string, strCodeFn func(srcName string) Code, errStatement func(arg, header string) *Statement) (block *Statement) {

	block = Line()
	if len(varMap) != 0 {
		for argName, srcName := range varMap {
			argName = strings.TrimPrefix(argName, "!")
			argTokens := strings.Split(argName, ".")
			argName = argTokens[0]
			argVarName := strings.Join(argTokens, "")
			vArg := m.argByName(argName)
			if vArg == nil {
				if m.resultByName(argName) == nil {
					m.log.WithField("svc", m.svc.Name).WithField("method", m.Name).WithField("arg", argVarName).WithField(typeName, srcName).Warning("result not found")
				}
				continue
			}
			argID := Id(argVarName)
			argType := vArg.Type
			argTypeName := argType.String()
			if len(argTokens) > 1 {
				argType = nestedType(vArg.Type, "", argTokens)
				argTypeName = argType.String()
			}
			switch t := argType.(type) { // nolint:gocritic
			case types.TPointer:
				argID = Op("&").Add(argID)
				argTypeName = t.NextType().String()
			}
			block.If(Id("_" + argVarName).Op(":=").Add(strCodeFn(srcName)).Op(";").Id("_" + argVarName).Op("!=").Lit("")).
				BlockFunc(func(g *Group) {
					if pkg := importPackage(argType); pkg != "" {
						g.Var().Id(argVarName).Qual(pkg, strings.Split(argTypeName, ".")[1])
					} else {
						g.Var().Id(argVarName).Id(argTypeName)
					}
					g.Add(m.argToTypeConverter(Id("_"+argVarName), argType, Id(argVarName), errStatement(argVarName, srcName)))
					reqID := g.Id("request").Dot(utils.ToCamel(argName))
					if len(argTokens) > 1 {
						for _, token := range argTokens[1:] {
							reqID = reqID.Dot(token)
						}
						reqID.Op("=").Add(argID)
						return
					}
					reqID.Op("=").Add(argID)
				}).Line()
		}
	}
	return
}

func (m *method) httpRetHeaders() (block *Statement) {

	block = Line()
	if len(m.varHeaderMap()) != 0 {
		for ret, header := range m.varHeaderMap() {
			vArg := m.resultByName(ret)
			if vArg == nil {
				if m.argByName(ret) == nil {
					m.log.WithField("svc", m.svc.Name).WithField("method", m.Name).WithField("ret", ret).WithField("header", header).Warning("result not found")
				}
				continue
			}
			block.Id(_ctx_).Dot("Set").Call(Lit(header), Qual(packageFmt, "Sprint").Call(Id("response").Dot(utils.ToCamel(ret))))
		}
	}
	return block
}

func (m *method) argParamMap() (params map[string]string) {

	if m.paramToArg != nil {
		return m.paramToArg
	}

	m.paramToArg = make(map[string]string)

	if urlArgs := m.tags.Value(tagHttpArg); urlArgs != "" {

		paramPairs := strings.Split(urlArgs, ",")

		for _, pair := range paramPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				param := strings.TrimSpace(pairTokens[1])
				m.paramToArg[arg] = param
			}
		}
	}
	return m.paramToArg
}

func (m *method) argPathMap() (paths map[string]string) {

	if m.pathToArg != nil {
		return m.pathToArg
	}

	m.pathToArg = make(map[string]string)

	if urlPath := m.tags.Value(tagHttpPath); urlPath != "" {

		urlTokens := strings.Split(urlPath, "/")

		for _, token := range urlTokens {
			if strings.HasPrefix(token, ":") {
				arg := strings.TrimSpace(strings.TrimPrefix(token, ":"))
				m.pathToArg[arg] = arg
			}
		}
	}
	return m.pathToArg
}

func (m *method) argCookieMap() (cookies map[string]string) {

	if m.cookieToArg != nil {
		return m.cookieToArg
	}

	m.cookieToArg = make(map[string]string)

	for varName, cookieName := range m.varCookieMap() {
		if m.argByName(varName) != nil {
			m.cookieToArg[varName] = cookieName
		}
	}
	return m.cookieToArg
}

func (m *method) retCookieMap() (cookies map[string]string) {

	if m.cookieToRet != nil {
		return m.cookieToRet
	}

	m.cookieToRet = make(map[string]string)

	for varName, cookieName := range m.varCookieMap() {
		if m.resultByName(varName) != nil {
			m.cookieToRet[varName] = cookieName
		}
	}
	return m.cookieToRet
}

func (m *method) varCookieMap() (cookies map[string]string) {

	if m.cookieToVar != nil {
		return m.cookieToVar
	}

	m.cookieToVar = make(map[string]string)

	if httpCookies := m.tags.Value(tagHttpCookies); httpCookies != "" {

		cookiePairs := strings.Split(httpCookies, ",")

		for _, pair := range cookiePairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				cookie := strings.TrimSpace(pairTokens[1])
				m.cookieToVar[arg] = cookie
			}
		}
	}
	return m.cookieToVar
}

func (m *method) varHeaderMap() (headers map[string]string) {

	if m.headerToVar != nil {
		return m.headerToVar
	}

	m.headerToVar = make(map[string]string)

	if httpHeaders := m.tags.Value(tagHttpHeader); httpHeaders != "" {

		headerPairs := strings.Split(httpHeaders, ",")

		for _, pair := range headerPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				header := strings.TrimSpace(pairTokens[1])
				m.headerToVar[arg] = header
			}
		}
	}
	return m.headerToVar
}

func (m *method) argByName(argName string) (variable *types.Variable) {

	argName = strings.TrimPrefix(argName, "!")
	for _, arg := range m.Args {
		if arg.Name == argName {
			return &arg
		}
	}
	return
}

func (m *method) resultByName(retName string) (variable *types.Variable) {

	for _, ret := range m.Results {
		if ret.Name == retName {
			return &ret
		}
	}
	return
}

func (m *method) fieldsResult() []types.StructField {
	return m.resultFields
}

func (m *method) fieldsArgument() []types.StructField {
	return m.argFields
}

func (m *method) resultsWithoutError() (vars []types.Variable) {

	if isErrorLast(m.Results) {
		return m.Results[:len(m.Results)-1]
	}
	return m.Results
}

func (m *method) resultFieldsWithoutError() (vars []types.Variable) {

	var resultVars []types.Variable
	if isErrorLast(m.Results) {
		resultVars = m.Results[:len(m.Results)-1]
	} else {
		resultVars = m.Results
	}
	for _, v := range resultVars {
		if m.isInlined(&v) {
		nextTick:
			switch vType := v.Type.(type) {
			case types.TPointer:
				v = types.Variable{Base: types.Base{Name: vType.Next.String()}, Type: vType.Next}
				goto nextTick
			case types.TImport:
				vars = append(vars, types.Variable{Base: types.Base{Name: vType.Next.String()}, Type: vType.Next})
				continue
			}
		}
		vars = append(vars, v)
	}
	return
}

func (m *method) argsWithoutContext() (args []types.Variable) {

	if isContextFirst(m.Args) {
		return m.Args[1:]
	}
	return m.Args
}

func (m *method) argsFieldsWithoutContext() (vars []types.Variable) {

	var argVars []types.Variable
	if isContextFirst(m.Args) {
		argVars = m.Args[1:]
	} else {
		argVars = m.Args
	}
	for _, v := range argVars {
		if m.isInlined(&v) {
			switch vType := v.Type.(type) { // nolint:gocritic
			case types.TImport:
				vars = append(vars, types.Variable{Base: types.Base{Name: vType.Next.String()}, Type: vType.Next})
				continue
			}
		}
		vars = append(vars, v)
	}
	return
}

func (m *method) argToTypeConverter(from *Statement, vType types.Type, id *Statement, errStatement *Statement) *Statement {

	op := "="

	uuidPackage := m.tags.Value(tagPackageUUID, packageUUID)

	typename := types.TypeName(vType)
	if typename == nil {
		panic("need to check and update validation rules (2)")
	}
	switch *typename {
	case "string":
		return id.Op(op).Add(from)
	case "bool":
		return List(id, Err()).Op(op).Qual(packageStrconv, "ParseBool").Call(from).Add(errStatement)
	case "int":
		return List(id, Err()).Op(op).Qual(packageStrconv, "Atoi").Call(from).Add(errStatement)
	case "int64":
		return List(id, Err()).Op(op).Qual(packageStrconv, "ParseInt").Call(from, Lit(10), Lit(64)).Add(errStatement)
	case "int32":
		return List(id, Err()).Op(op).Qual(packageStrconv, "ParseInt").Call(from, Lit(10), Lit(32)).Add(errStatement)
	case "uint", "uint64":
		return List(id, Err()).Op(op).Qual(packageStrconv, "ParseUint").Call(from, Lit(10), Lit(64)).Add(errStatement)
	case "uint32":
		return List(id, Err()).Op(op).Qual(packageStrconv, "ParseUint").Call(from, Lit(10), Lit(32)).Add(errStatement)
	case "float64":
		return List(id, Err()).Op(op).Qual(packageStrconv, "ParseFloat").Call(from, Lit(64)).Add(errStatement)
	case "float32":
		temp64 := Id("temp64")
		return List(temp64, Err()).Op(":=").Qual(packageStrconv, "ParseFloat").Call(from, Lit(32)).Add(errStatement).Add(Line()).Add(id.Op(op).Float32().Call(temp64))
	case "UUID":
		return List(id, Id("_")).Op(op).Qual(uuidPackage, "Parse").Call(from)

	case "Time":
		return List(id, Err()).Op(op).Qual(packageTime, "Parse").Call(Qual(packageTime, "RFC3339Nano"), from).Add(errStatement)
	default:
		return Op("_").Op("=").Qual(m.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Op("[]").Byte().Call(Op("`\"`").Op("+").Add(from).Op("+").Op("`\"`")), Op("&").Add(id))
	}
}

func (m *method) varsToFields(vars []types.Variable, tags tags.DocTags, excludes ...map[string]string) (fields []types.StructField) {

	for _, variable := range vars {

		field := types.StructField{Variable: variable, Tags: make(map[string][]string)}

		for _, exclude := range excludes {
			for varName := range exclude {
				if strings.HasPrefix(varName, "!") {
					if variable.Name == varName[1:] {
						field.Tags["json"] = []string{"-"}
					}
				}
			}
		}
		for key, value := range tags.Sub(variable.Name) {
			if key == tagTag {
				if list := strings.Split(value, "|"); len(list) > 0 {
					for _, item := range list {
						if tokens := strings.Split(item, ":"); len(tokens) == 2 {
							if tokens[1] == "inline" {
								tokens[1] = ",inline"
							}
							field.Tags[tokens[0]] = strings.Split(tokens[1], ",")
						}
					}
				}
			}
		}
		fields = append(fields, field)
	}
	return
}

func (m *method) isInlined(field *types.Variable) (isInlined bool) {

	for key, value := range m.tags.Sub(field.Name) {
		if key == tagTag {
			if list := strings.Split(value, "|"); len(list) > 0 {
				for _, item := range list {
					if tokens := strings.Split(item, ":"); len(tokens) == 2 {
						if tokens[0] == "json" {
							for _, json := range strings.Split(tokens[1], ",") {
								if json == "inline" {
									return true
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

func importPackage(varType types.Type) string {

	switch vType := varType.(type) {
	case types.TImport:
		return vType.Import.Package
	case types.TPointer:
		return importPackage(vType.Next)
	}
	return ""
}
