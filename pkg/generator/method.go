// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (method.go at 18.06.2020, 12:27) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vetcher/go-astra/types"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/pkg/tags"
	"github.com/seniorGolang/tg/pkg/utils"
)

type method struct {
	*types.Function

	log logrus.FieldLogger

	svc  *service
	tags tags.DocTags

	uploadVars   map[string]string
	downloadVars map[string]string

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
		tags:     tags.ParseTags(fn.Docs),
	}
	m.argFields = m.varsToFields(m.argsWithoutContext(), m.tags, m.argCookieMap())
	m.resultFields = m.varsToFields(m.resultsWithoutError(), m.tags, m.retCookieMap(), m.varHeaderMap())
	return
}

func (m method) lcName() string {
	return strings.ToLower(m.Name)
}

func (m method) lccName() string {
	return utils.ToLowerCamel(m.Name)
}

func (m method) requestStructName() string {
	return "request" + m.svc.Name + m.Name
}

func (m method) responseStructName() string {
	return "response" + m.svc.Name + m.Name
}

func (m method) isUploadVar(varName string) bool {
	_, found := m.uploadVarsMap()[varName]
	return found
}

func (m method) isDownloadVar(varName string) bool {
	_, found := m.downloadVarsMap()[varName]
	return found
}

func (m *method) uploadVarsMap() (headers map[string]string) {

	if m.uploadVars != nil {
		return m.uploadVars
	}

	m.uploadVars = make(map[string]string)

	if uploadVars := m.tags.Value(tagUploadVars); uploadVars != "" {

		uploadPairs := strings.Split(uploadVars, ",")

		for _, pair := range uploadPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				upload := strings.TrimSpace(pairTokens[1])
				m.uploadVars[arg] = upload
			}
		}
	}
	return m.uploadVars
}

func (m *method) downloadVarsMap() (headers map[string]string) {

	if m.downloadVars != nil {
		return m.uploadVars
	}

	m.downloadVars = make(map[string]string)

	if uploadVars := m.tags.Value(tagDownloadVars); uploadVars != "" {

		downloadPairs := strings.Split(uploadVars, ",")

		for _, pair := range downloadPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				upload := strings.TrimSpace(pairTokens[1])
				m.downloadVars[arg] = upload
			}
		}
	}
	return m.downloadVars
}

func (m method) httpPath(withoutPrefix ...bool) string {
	var elements []string
	if len(withoutPrefix) == 0 {
		elements = append(elements, "/")
	}
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName()))
	return path.Join(append(elements, prefix, urlPath)...)
}

func (m method) jsonrpcPath(withoutPrefix ...bool) string {
	var elements []string
	if len(withoutPrefix) == 0 {
		elements = append(elements, "/")
	}
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName()))
	return path.Join(append(elements, prefix, urlPath)...)
}

func (m method) httpMethod() string {
	return strings.ToUpper(m.tags.Value(tagMethodHTTP, "POST"))
}

func (m method) isHTTP() bool {
	return m.svc.tags.Contains(tagServerHTTP) && m.tags.Contains(tagMethodHTTP)
}

func (m method) isJsonRPC() bool {
	return m.svc.tags.Contains(tagServerJsonRPC) && !m.tags.Contains(tagMethodHTTP)
}

func (m method) handlerQual() (pkgPath, handler string) {

	if !m.tags.Contains(tagHandler) {
		return
	}
	if tokens := strings.Split(m.tags.Value(tagHandler), ":"); len(tokens) == 2 {
		return tokens[0], tokens[1]
	}
	return
}

func (m method) arguments() (vars []types.StructField) {

	argsAll := m.fieldsArgument()

	for _, arg := range argsAll {

		_, inPath := m.argPathMap()[arg.Name]
		_, inArgs := m.argParamMap()[arg.Name]
		_, inHeader := m.varHeaderMap()[arg.Name]
		_, inCookie := m.varCookieMap()[arg.Name]
		_, inUpload := m.uploadVarsMap()[arg.Name]

		if !inArgs && !inPath && !inHeader && !inCookie && !inUpload {

			if m.isUploadVar(arg.Name) {
				m.tags.Set(arg.Name+".type", "file")
				m.tags.Set(arg.Name+".format", "byte")
			}
			vars = append(vars, arg)
		}
	}
	return
}

func (m method) argumentsWithUploads() (vars []types.StructField) {

	argsAll := m.fieldsArgument()

	for _, arg := range argsAll {

		_, inPath := m.argPathMap()[arg.Name]
		_, inArgs := m.argParamMap()[arg.Name]
		_, inHeader := m.varHeaderMap()[arg.Name]
		_, inCookie := m.varCookieMap()[arg.Name]

		if !inArgs && !inPath && !inHeader && !inCookie {

			if m.isUploadVar(arg.Name) {
				m.tags.Set(arg.Name+".type", "file")
				m.tags.Set(arg.Name+".format", "byte")
			}
			if jsonTags, _ := arg.Tags["json"]; len(jsonTags) == 0 {
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

func (m method) results() (vars []types.StructField) {

	argsAll := m.fieldsResult()

	for _, arg := range argsAll {

		_, inHeader := m.varHeaderMap()[arg.Name]
		_, inCookie := m.varCookieMap()[arg.Name]
		_, inDownload := m.downloadVarsMap()[arg.Name]

		if !inHeader && !inCookie && !inDownload {
			arg.Tags = map[string][]string{"json": {arg.Name}}
			arg.Name = utils.ToCamel(arg.Name)
			vars = append(vars, arg)
		}
	}
	return
}

func (m method) urlArgs(errStatement func(arg, header string) *Statement) (g *Statement) {

	return m.argFromString("urlParam", m.argPathMap(),
		func(srcName string) Code {
			return Id(_ctx_).Dot("UserValue").Call(Lit(srcName)).Op(".").Call(String())
		},
		errStatement,
	)
}

func (m method) urlParams(errStatement func(arg, header string) *Statement) (block *Statement) {

	return m.argFromString("urlParam", m.argParamMap(),
		func(srcName string) Code {
			return Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("QueryArgs").Call().Dot("Peek").Call(Lit(srcName)))
		},
		errStatement,
	)
}

func (m method) httpArgHeaders(errStatement func(arg, header string) *Statement) (block *Statement) {

	return m.argFromString("header", m.varHeaderMap(),
		func(srcName string) Code {
			return Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Request").Dot("Header").Dot("Peek").Call(Lit(srcName)))
		},
		errStatement,
	)
}

func (m method) httpCookies(errStatement func(arg, header string) *Statement) (block *Statement) {

	return m.argFromString("cookie", m.varCookieMap(),
		func(srcName string) Code {
			return Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Request").Dot("Header").Dot("Cookie").Call(Lit(srcName)))
		},
		errStatement,
	)
}

func (m method) argFromString(typeName string, varMap map[string]string, strCodeFn func(srcName string) Code, errStatement func(arg, header string) *Statement) (block *Statement) {

	block = Line()
	if len(varMap) != 0 {
		for argName, srcName := range varMap {
			argTokens := strings.Split(argName, ".")
			argName = argTokens[0]
			argVarName := strings.Join(argTokens, "")
			vArg := m.argByName(argName)
			if vArg == nil {
				if m.resultByName(argName) == nil {
					m.log.WithField("svc", m.svc.Name).WithField("method", m.Name).WithField("arg", argVarName).WithField(typeName, srcName).Warning("argument not found")
				}
				continue
			}
			argID := Id(argVarName)
			argType := vArg.Type
			argTypeName := argType.String()
			if len(argTokens) > 1 {
				argType = nestedType(vArg.Type, "", argTokens)
			}
			switch t := argType.(type) {
			case types.TPointer:
				argID = Op("&").Add(argID)
				argTypeName = t.NextType().String()
			}
			block.If(Id("_" + argVarName).Op(":=").Add(strCodeFn(srcName)).Op(";").Id("_" + argVarName).Op("!=").Lit("")).
				BlockFunc(func(g *Group) {
					g.Var().Id(argVarName).Id(argTypeName)
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

func (m method) httpRetHeaders() (block *Statement) {

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
			block.If(Id("response").Dot(utils.ToCamel(ret)).Op("!=").Lit("").Block(
				Id(_ctx_).Dot("Response").Dot("Header").Dot("Set").Call(Lit(header), Id("response").Dot(utils.ToCamel(ret))),
			))
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
			if strings.HasPrefix(token, "{") {
				arg := strings.TrimSpace(strings.Replace(strings.TrimPrefix(token, "{"), "}", "", -1))
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

func (m method) argByName(argName string) (variable *types.Variable) {

	for _, arg := range m.Args {
		if arg.Name == argName {
			return &arg
		}
	}
	return
}

func (m method) resultByName(retName string) (variable *types.Variable) {

	for _, ret := range m.Results {
		if ret.Name == retName {
			return &ret
		}
	}
	return
}

func (m method) fieldsResult() []types.StructField {
	return m.resultFields
}

func (m method) fieldsArgument() []types.StructField {
	return m.argFields
}

func (m method) resultsWithoutError() []types.Variable {
	if isErrorLast(m.Results) {
		return m.Results[:len(m.Results)-1]
	}
	return m.Results
}

func (m method) argsWithoutContext() (args []types.Variable) {

	if isContextFirst(m.Args) {
		return m.Args[1:]
	}
	return m.Args
}

func (m method) argToTypeConverter(from *Statement, vType types.Type, id *Statement, errStatement *Statement) *Statement {

	op := "="

	uuidPackage := m.tags.Value(tagPackageUUID, packageUUID)

	typename := types.TypeName(vType)
	if typename == nil {
		panic("need to check and update validation rules (2)")
	}
	switch *typename {
	case "string":
		return id.Op(op).Add(from)
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
	case "UUID":
		return id.Op(op).Qual(uuidPackage, "FromStringOrNil").Call(from)

	case "Time":
		return List(id, Err()).Op(op).Qual(packageTime, "Parse").Call(Qual(packageTime, "RFC3339Nano"), from).Add(errStatement)
	}
	return Line().Add(from)
}

func (m method) varsToFields(vars []types.Variable, tags tags.DocTags, excludes ...map[string]string) (fields []types.StructField) {

	for _, variable := range vars {

		field := types.StructField{Variable: variable, Tags: make(map[string][]string)}

		for _, exclude := range excludes {
			if _, found := exclude[variable.Name]; found {
				field.Tags["json"] = []string{"-"}
			}
		}
		for key, value := range tags.Sub(variable.Name) {

			if key == tagTag {
				if list := strings.Split(value, "|"); len(list) > 0 {
					for _, item := range list {
						if tokens := strings.Split(item, ":"); len(tokens) == 2 {
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
