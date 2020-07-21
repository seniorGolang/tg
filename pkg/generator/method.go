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
	headerToArg map[string]string
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
	m.resultFields = m.varsToFields(m.resultsWithoutError(), m.tags, m.retCookieMap())
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

func (m method) httpPath() string {
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName()))
	return path.Join("/", prefix, urlPath)
}

func (m method) jsonrpcPath() string {
	prefix := m.svc.tags.Value(tagHttpPrefix)
	urlPath := m.tags.Value(tagHttpPath, path.Join("/", m.svc.lccName(), m.lccName()))
	return path.Join("/", prefix, urlPath)
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

func (m method) urlArgs(errStatement *Statement) (g *Statement) {

	g = Line()
	if len(m.argPathMap()) != 0 {
		for arg, param := range m.argPathMap() {
			vArg := m.argByName(arg)
			if vArg == nil {
				m.log.WithField("svc", m.svc.Name).WithField("method", m.Name).WithField("arg", arg).WithField("param", param).Warning("argument not found")
				continue
			}
			g.Line().List(Id("_"+vArg.Name), Id("_")).Op(":=").Id(_ctx_).Dot("UserValue").Call(Lit(param)).Op(".").Call(String())
			g.Line().Add(m.argToTypeConverter(Id("_"+vArg.Name), vArg.Type, Id(
				"request").Dot(utils.ToCamel(vArg.Name)), errStatement)).Line()
		}
	}
	return
}

func (m method) arguments() (vars []types.StructField) {

	argsAll := m.fieldsArgument()

	for _, arg := range argsAll {

		_, inPath := m.argPathMap()[arg.Name]
		_, inArgs := m.argParamMap()[arg.Name]
		_, inHeader := m.argHeaderMap()[arg.Name]
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
		_, inHeader := m.argHeaderMap()[arg.Name]
		_, inCookie := m.varCookieMap()[arg.Name]

		if !inArgs && !inPath && !inHeader && !inCookie {

			if m.isUploadVar(arg.Name) {
				m.tags.Set(arg.Name+".type", "file")
				m.tags.Set(arg.Name+".format", "byte")
			}
			arg.Tags = map[string][]string{"json": {arg.Name}}
			arg.Name = utils.ToCamel(arg.Name)
			vars = append(vars, arg)
		}
	}
	return
}

func (m method) results() (vars []types.StructField) {

	argsAll := m.fieldsResult()

	for _, arg := range argsAll {

		_, inHeader := m.argHeaderMap()[arg.Name]
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

func (m method) urlParams(errStatement *Statement) (g *Statement) {

	g = Line()

	if len(m.argParamMap()) != 0 {

		for arg, param := range m.argParamMap() {

			vArg := m.argByName(arg)

			if types.IsArray(vArg.Type) || types.IsEllipsis(vArg.Type) {

				vArg := m.argByName(param)

				if vArg == nil {
					m.log.WithField("svc", m.svc.Name).WithField("method", m.Name).WithField("arg", arg).WithField("path", param).Warning("argument not found")
					continue
				}

				var vType = vArg.Type

				if types.IsArray(vArg.Type) {
					vType = vArg.Type.(types.TArray).Next
				}

				if types.IsEllipsis(vArg.Type) {
					vType = vArg.Type.(types.TEllipsis).Next
				}

				g.Line().Id("arr" + utils.ToCamel(param)).Op(":=").Id(_ctx_).Dot("QueryArgs").Call().Dot("PeekMulti").Call(Lit(param))

				g.Line().For(List(Id("_"), Id("param")).Op(":=").Range().Id("arr"+utils.ToCamel(param))).Block(

					Line().Var().Id("_"+vArg.Name).Id(vType.String()),
					Add(m.argToTypeConverter(Qual(packageGotils, "B2S").Call(Id("param")), vType, Id("_"+param), errStatement)),
					Line().Id("request").Dot(utils.ToCamel(vArg.Name)).Op("=").Append(Id("request").Dot(utils.ToCamel(vArg.Name)), Id("_"+vArg.Name)).Line(),
				)
				break
			}
			g.Line().Id("_"+vArg.Name).Op(":=").Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("QueryArgs").Call().Dot("Peek").Call(Lit(param)))
			g.Line().Add(m.argToTypeConverter(Id("_"+vArg.Name), vArg.Type, Id("request").Dot(utils.ToCamel(vArg.Name)), errStatement)).Line()
		}
	}
	return g
}

func (m method) httpHeaders(errStatement func(arg, header string) *Statement) (block *Statement) {

	block = Line()
	if len(m.argHeaderMap()) != 0 {

		for arg, header := range m.argHeaderMap() {

			vArg := m.argByName(arg)
			if vArg == nil {
				m.log.WithField("svc", m.svc.Name).WithField("method", m.Name).WithField("arg", arg).WithField("header", header).Warning("argument not found")
				continue
			}
			block.If(Id("_"+arg).Op(":=").Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Request").Dot("Header").Dot("Peek").Call(Lit(header))).Op(";").Id("_" + arg).Op("!=").Lit("")).Block(
				Add(m.argToTypeConverter(Id("_"+vArg.Name), vArg.Type, Id("request").Dot(utils.ToCamel(arg)), errStatement(arg, header))),
			).Line()
		}
	}
	return block
}

func (m method) httpCookies(errStatement func(arg, header string) *Statement) (block *Statement) {

	block = Line()
	if len(m.argCookieMap()) != 0 {

		for arg, header := range m.argCookieMap() {
			vArg := m.argByName(arg)
			block.If(Id("_"+arg).Op(":=").Qual(packageGotils, "B2S").Call(Id(_ctx_).Dot("Request").Dot("Header").Dot("Cookie").Call(Lit(header))).Op(";").Id("_" + arg).Op("!=").Lit("")).Block(
				Add(m.argToTypeConverter(Id("_"+vArg.Name), vArg.Type, Id("request").Dot(utils.ToCamel(arg)), errStatement(arg, header))),
			).Line()
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

func (m *method) argHeaderMap() (headers map[string]string) {

	if m.headerToArg != nil {
		return m.headerToArg
	}

	m.headerToArg = make(map[string]string)

	if httpHeaders := m.tags.Value(tagHttpHeader); httpHeaders != "" {

		headerPairs := strings.Split(httpHeaders, ",")

		for _, pair := range headerPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				header := strings.TrimSpace(pairTokens[1])
				m.headerToArg[arg] = header
			}
		}
	}
	return m.headerToArg
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

func (m method) varsToFields(vars []types.Variable, tags tags.DocTags, cookiesMap map[string]string) (fields []types.StructField) {

	for _, variable := range vars {

		field := types.StructField{Variable: variable, Tags: make(map[string][]string)}

		if _, found := cookiesMap[variable.Name]; found {
			field.Tags["json"] = []string{"-"}
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
