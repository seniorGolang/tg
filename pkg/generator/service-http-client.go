// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (service-rest.go at 23.06.2020, 23:36) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
	"github.com/seniorGolang/tg/v2/pkg/tags"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

func (svc *service) renderClientHTTP(outDir string) (err error) {

	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint

	if err = pkgCopyTo("httpclient", outDir); err != nil {
		return err
	}
	srcFile.ImportName(packageContext, "context")
	srcFile.ImportName(packageFmt, "fmt")
	srcFile.ImportName(packageTime, "time")
	srcFile.ImportName(packageUUID, "goUUID")
	srcFile.ImportName(packageFiber, "fiber")

	srcFile.ImportName(fmt.Sprintf("%s/cache", svc.tr.pkgPath(outDir)), "cache")
	srcFile.ImportName(fmt.Sprintf("%s/hasher", svc.tr.pkgPath(outDir)), "hasher")
	srcFile.ImportName(fmt.Sprintf("%s/httpclient", svc.tr.pkgPath(outDir)), "httpclient")

	srcFile.Type().Id("Client" + svc.Name).StructFunc(func(g *Group) {
		g.Id("httpClient").Op("*").Qual(fmt.Sprintf("%s/httpclient", svc.tr.pkgPath(outDir)), "ClientHTTP")
	}).Line()

	srcFile.Func().Id("NewClient"+svc.Name).Params(
		Id("endpoint").String(),
		Id("opts").Op("...").Qual(fmt.Sprintf("%s/httpclient", svc.tr.pkgPath(outDir)), "Option"),
	).Params(Id("client").Op("*").Id("Client"+svc.Name)).Block(
		List(Id("httpClient")).Op(":=").Qual(fmt.Sprintf("%s/httpclient", svc.tr.pkgPath(outDir)), "NewClient").Call(Id("endpoint"), Id("opts").Op("...")),
		Return(Op("&").Id("Client"+svc.Name).Values(Dict{
			Id("httpClient"): Id("httpClient"),
		})),
	).Line()

	for _, method := range svc.methods {
		srcFile.Line().Add(svc.httpClientMethodFunc(ctx, method, outDir))
	}
	return srcFile.Save(path.Join(outDir, svc.lcName()+"-http-client.go"))
}

func (svc *service) httpClientMethodFunc(ctx context.Context, method *method, _ string) Code {

	c := Comment(fmt.Sprintf("%s performs the %s operation.", method.Name, method.Name))
	c.Line()
	c.Func().Params(Id("cli").Op("*").Id("Client" + svc.Name)).
		Id(method.Name).
		Params(funcDefinitionParams(ctx, method.Args)).
		Params(funcDefinitionParams(ctx, method.Results)).
		BlockFunc(func(g *Group) {
			g.Line()
			g.Var().Id("reqBody").Index().Byte()
			var httpMethod string
			if method.tags.Contains(tagMethodHTTP) {
				httpMethod = method.tags.Value(tagMethodHTTP)
			} else {
				httpMethod = "POST"
			}
			var successStatusCode int
			if method.tags.Contains(tagHttpSuccess) {
				successCodeStr := method.tags.Value(tagHttpSuccess)
				code, err := strconv.Atoi(successCodeStr)
				if err != nil {
					successStatusCode = http.StatusOK
				} else {
					successStatusCode = code
				}
			} else {
				successStatusCode = http.StatusOK
			}
			methodPath := strings.ToLower(method.Name)
			pathParams := make(map[string]string)
			if method.tags.Contains(tagHttpPath) {
				methodPathAnnotation := method.tags.Value(tagHttpPath)
				matches := argPathMap(method.tags)
				for key, value := range matches {
					placeholder := value
					paramName := key
					pathParams[placeholder] = paramName
				}
				methodPath = methodPathAnnotation
			}
			svcPrefix := svc.tags.Value(tagHttpPrefix, svc.Name)
			argsMappings := varArgsMap(method.tags)
			cookieMappings := varCookieMap(method.tags)
			headerMappings := varHeaderMap(method.tags)
			if len(method.arguments()) > 1 {
				g.Id("request").Op(":=").Id(method.requestStructName()).Values(DictFunc(func(dict Dict) {
					for idx, arg := range method.argsWithoutContext() {
						if _, exists := argsMappings[arg.Name]; exists {
							continue
						}
						if _, exists := cookieMappings[arg.Name]; exists {
							continue
						}
						if _, exists := headerMappings[arg.Name]; exists {
							continue
						}
						if _, exists := pathParams[arg.Name]; exists {
							continue
						}
						dict[Id(utils.ToCamel(arg.Name))] = Id(method.argsWithoutContext()[idx].Name)
					}
				}))
				g.Id("reqBody").Op(",").Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Marshal").Call(Id("request"))
				g.If(Err().Op("!=").Nil()).Block(
					Return(),
				)
			}
			g.Id("req").Op(":=").Qual(packageFasthttp, "AcquireRequest").Call()
			g.Defer().Qual(packageFasthttp, "ReleaseRequest").Call(Id("req"))
			urlPathFmt := methodPath
			for placeholder := range pathParams {
				urlPathFmt = strings.ReplaceAll(urlPathFmt, ":"+placeholder, "%v")
			}
			fullURLPath := path.Join("%s", svcPrefix, urlPathFmt)
			var urlPathArgs []Code
			urlPathArgs = append(urlPathArgs, Lit(fullURLPath))
			urlPathArgs = append(urlPathArgs, Id("cli").Dot("httpClient").Dot("BaseURL"))
			for _, paramName := range pathParams {
				urlPathArgs = append(urlPathArgs, Id(paramName))
			}
			g.Id("req").Dot("SetRequestURI").Call(
				Qual(packageFmt, "Sprintf").Call(urlPathArgs...),
			)
			g.Id("req").Dot("Header").Dot("SetMethod").Call(Lit(httpMethod))
			g.Id("req").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit("application/json"))
			g.Id("req").Dot("SetBody").Call(Id("reqBody"))
			for paramName, cookieName := range cookieMappings {
				g.Id("req").Dot("Header").Dot("SetCookie").Call(Lit(cookieName), varToString(method.argByName(paramName)))
			}
			for paramName, headerName := range headerMappings {
				g.Id("req").Dot("Header").Dot("Set").Call(Lit(headerName), varToString(method.argByName(paramName)))
			}
			for paramName, argName := range argsMappings {
				paramVar := method.argByName(paramName)
				if isPointerType(paramVar.Type) {
					g.If(Id(paramName).Op("!=").Nil()).Block(
						Id("req").Dot("URI").Call().Dot("QueryArgs").Call().Dot("Set").Call(Lit(argName), varToString(paramVar)),
					)
				} else {
					g.Id("req").Dot("URI").Call().Dot("QueryArgs").Call().Dot("Set").Call(Lit(argName), varToString(paramVar))
				}
			}
			g.Id("resp").Op(":=").Qual(packageFasthttp, "AcquireResponse").Call()
			g.Defer().Qual(packageFasthttp, "ReleaseResponse").Call(Id("resp"))
			g.If(List(Id("deadline"), Id("ok")).Op(":=").Id(_ctx_).Dot("Deadline").Call(), Id("ok")).Block(
				Id("timeout").Op(":=").Qual(packageTime, "Until").Call(Id("deadline")),
				Id("cli").Dot("httpClient").Dot("SetTimeout").Call(Id("timeout")),
			)
			g.If(Err().Op("=").Id("cli").Dot("httpClient").Dot("Do").Call(Id("ctx"), Id("req"), Id("resp")).Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			)
			g.Id("respBody").Op(":=").Id("resp").Dot("Body").Call()
			g.If(Id("resp").Dot("StatusCode").Call().Op("!=").Lit(successStatusCode)).Block(
				Err().Op("=").Qual(packageFmt, "Errorf").Call(
					Lit("HTTP error: %d. URL: %s, Method: %s, Body: %s"),
					Id("resp").Dot("StatusCode").Call(),
					Id("req").Dot("URI").Call().Dot("String").Call(),
					Id("req").Dot("Header").Dot("Method").Call(),
					String().Call(Id("respBody")),
				),
				Return(),
			)
			if len(method.resultsWithoutError()) == 1 {
				g.Var().Id("response").Id(method.responseStructName())
				g.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id("respBody"), Op("&").Id("response").Dot(utils.ToCamel(method.resultsWithoutError()[0].Name))).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				for _, ret := range method.resultsWithoutError() {
					g.Id(ret.Name).Op("=").Id("response").Dot(utils.ToCamel(ret.Name))
				}
			} else {
				g.Var().Id("response").Id(method.responseStructName())
				g.If(Err().Op("=").Qual(svc.tr.tags.Value(tagPackageJSON, packageStdJSON), "Unmarshal").Call(Id("respBody"), Op("&").Id("response")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				for _, ret := range method.resultsWithoutError() {
					g.Id(ret.Name).Op("=").Id("response").Dot(utils.ToCamel(ret.Name))
				}
			}
			g.Return()
		})
	return c
}

func argPathMap(tags tags.DocTags) (paths map[string]string) {

	pathToArg := make(map[string]string)
	if urlPath := tags.Value(tagHttpPath); urlPath != "" {
		urlTokens := strings.Split(urlPath, "/")
		for _, token := range urlTokens {
			if strings.HasPrefix(token, ":") {
				arg := strings.TrimSpace(strings.TrimPrefix(token, ":"))
				pathToArg[arg] = arg
			}
		}
	}
	return pathToArg
}

func varArgsMap(tags tags.DocTags) map[string]string {

	cookieToVar := make(map[string]string)
	if httpCookies := tags.Value(tagHttpArg); httpCookies != "" {
		cookiePairs := strings.Split(httpCookies, ",")
		for _, pair := range cookiePairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				cookie := strings.TrimSpace(pairTokens[1])
				cookieToVar[arg] = cookie
			}
		}
	}
	return cookieToVar
}

func varCookieMap(tags tags.DocTags) map[string]string {

	cookieToVar := make(map[string]string)
	if httpCookies := tags.Value(tagHttpCookies); httpCookies != "" {
		cookiePairs := strings.Split(httpCookies, ",")
		for _, pair := range cookiePairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				cookie := strings.TrimSpace(pairTokens[1])
				cookieToVar[arg] = cookie
			}
		}
	}
	return cookieToVar
}

func varHeaderMap(tags tags.DocTags) (headers map[string]string) {

	headerToVar := make(map[string]string)
	if httpHeaders := tags.Value(tagHttpHeader); httpHeaders != "" {
		headerPairs := strings.Split(httpHeaders, ",")
		for _, pair := range headerPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				header := strings.TrimSpace(pairTokens[1])
				headerToVar[arg] = header
			}
		}
	}
	return headerToVar
}

func varToString(variable *types.Variable) (code *Statement) {

	typename := types.TypeName(variable.Type)
	switch *typename {
	case "string":
		return Id(variable.Name)
	default:
		return Qual(packageFmt, "Sprint").Call(Id(variable.Name))
	}
}
