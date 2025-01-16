// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (swagger.go at 25.06.2020, 0:38) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v3"

	"github.com/seniorGolang/tg/v2/pkg/tags"
	"github.com/seniorGolang/tg/v2/pkg/utils"
)

const (
	contentJSON          = "application/json"
	bearerSecuritySchema = "bearer"
)

type swagger struct {
	*Transport

	schemas    swSchemas
	knownCount map[string]int
	knownTypes map[string]swSchema
}

func newSwagger(tr *Transport) (doc *swagger) {

	doc = &swagger{
		Transport:  tr,
		schemas:    make(swSchemas),
		knownCount: make(map[string]int),
		knownTypes: make(map[string]swSchema),
	}
	return
}

func (doc *swagger) render(outFilePath string, ifaces ...string) (err error) {

	var include, exclude = make([]string, 0, len(ifaces)), make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		if strings.HasPrefix(iface, "!") {
			exclude = append(exclude, strings.TrimPrefix(iface, "!"))
			continue
		}
		include = append(include, iface)
	}
	if len(include) != 0 && len(exclude) != 0 {
		err = fmt.Errorf("include and exclude cannot be set at same time (%v | %v)", include, exclude)
		return
	}
	if err = os.MkdirAll(filepath.Dir(outFilePath), 0777); err != nil {
		return
	}

	var swaggerDoc swObject
	swaggerDoc.OpenAPI = "3.0.0"
	swaggerDoc.Info.Title = doc.tags.Value(tagTitle)
	swaggerDoc.Info.Version = doc.tags.Value(tagAppVersion)
	swaggerDoc.Info.Description = doc.tags.Value(tagDesc)
	swaggerDoc.Paths = make(map[string]swPath)
	if doc.tags.IsSet(tagSecurity) {
		for _, security := range strings.Split(doc.tags.Value(tagSecurity), "|") {
			if strings.EqualFold(security, bearerSecuritySchema) {
				swaggerDoc.Security = append(swaggerDoc.Security, swSecurity{BearerAuth: []interface{}{}})
				swaggerDoc.Components.SecuritySchemes = &swSecuritySchemes{
					BearerAuth: swBearerAuth{
						Type:   "http",
						Scheme: security,
					},
				}
			}
		}
	}
	tagServers := strings.Split(doc.tags.Value(tagServers), "|")
	for _, tagServer := range tagServers {
		var serverDesc string
		serverValues := strings.Split(tagServer, ";")

		if len(serverValues) > 1 {
			serverDesc = serverValues[1]
		}
		swaggerDoc.Servers = append(swaggerDoc.Servers, swServer{URL: serverValues[0], Description: serverDesc})
	}
services:
	for _, serviceName := range doc.serviceKeys() {
		if len(include) != 0 {
			if !slices.Contains(include, serviceName) {
				doc.log.WithField("iface", serviceName).Info("skip")
				continue services
			}
		}
		if len(exclude) != 0 {
			if slices.Contains(exclude, serviceName) {
				doc.log.WithField("iface", serviceName).Info("skip")
				continue services
			}
		}
		service := doc.services[serviceName]
		serviceTags := strings.Split(service.tags.Value(tagSwaggerTags, service.Name), ",")
		doc.log.WithField("module", "swagger").Infof("service %s append jsonRPC methods", serviceTags)
		for _, method := range service.methods {
			if method.tags.Contains(tagSwaggerTags) {
				serviceTags = strings.Split(method.tags.Value(tagSwaggerTags), ",")
			}
			successCode := method.tags.ValueInt(tagHttpSuccess, fasthttp.StatusOK)

			doc.registerStruct(method.requestStructName(), service.pkgPath, method.tags, method.arguments())
			doc.registerStruct(method.responseStructName(), service.pkgPath, method.tags, method.results())

			var parameters []swParameter
			var retHeaders map[string]swHeader
			for argName, headerKey := range method.varHeaderMap() {
				if arg := method.argByName(argName); arg != nil {
					parameters = append(parameters, swParameter{
						In:       "header",
						Name:     headerKey,
						Required: true,
						Schema:   doc.walkVariable(arg.Name, service.pkgPath, arg.Type, nil),
					})
				}
				if ret := method.resultByName(argName); ret != nil {
					if retHeaders == nil {
						retHeaders = make(map[string]swHeader)
					}
					retHeaders[headerKey] = swHeader{
						Schema: doc.walkVariable(ret.Name, service.pkgPath, ret.Type, nil),
					}
				}
			}
			for argName, headerKey := range method.argPathMap() {
				if arg := method.argByName(argName); arg != nil {
					parameters = append(parameters, swParameter{
						In:       "path",
						Name:     headerKey,
						Required: true,
						Schema:   doc.walkVariable(arg.Name, service.pkgPath, arg.Type, nil),
					})
				}
				if ret := method.resultByName(argName); ret != nil {
					if retHeaders == nil {
						retHeaders = make(map[string]swHeader)
					}
					retHeaders[headerKey] = swHeader{
						Schema: doc.walkVariable(ret.Name, service.pkgPath, ret.Type, nil),
					}
				}
			}
			for argName, queryName := range method.argParamMap() {
				if arg := method.argByName(argName); arg != nil {
					parameters = append(parameters, swParameter{
						In:     "query",
						Name:   queryName,
						Schema: doc.walkVariable(arg.Name, service.pkgPath, arg.Type, nil),
					})
				}
			}
			for argName, cookieName := range method.varCookieMap() {
				if arg := method.argByName(argName); arg != nil {
					parameters = append(parameters, swParameter{
						In:       "cookie",
						Name:     cookieName,
						Required: true,
						Schema:   doc.walkVariable(arg.Name, service.pkgPath, arg.Type, nil),
					})
				}
				if ret := method.resultByName(argName); ret != nil {

					if retHeaders == nil {
						retHeaders = make(map[string]swHeader)
					}
					retHeaders["Set-Cookie"] = swHeader{
						Description: cookieName,
						Schema:      doc.walkVariable(ret.Name, service.pkgPath, ret.Type, nil),
					}
				}
			}
			if service.tags.Contains(tagServerJsonRPC) && !method.tags.Contains(tagMethodHTTP) {
				postMethod := &swOperation{
					Summary:     method.tags.Value(tagSummary),
					Description: method.tags.Value(tagDesc),
					Parameters:  parameters,
					Tags:        serviceTags,
					Deprecated:  method.tags.Contains(tagDeprecated),
					RequestBody: &swRequestBody{
						Content: swContent{
							contentJSON: swMedia{Schema: jsonrpcSchema("params", swSchema{Ref: "#/components/schemas/" + method.requestStructName()})},
						},
					},
					Responses: swResponses{
						"200": swResponse{
							Description: codeToText(200),
							Headers:     retHeaders,
							Content: swContent{
								contentJSON: swMedia{Schema: swSchema{
									OneOf: []swSchema{
										jsonrpcSchema("result", swSchema{Ref: "#/components/schemas/" + method.responseStructName()}),
										jsonrpcErrorSchema(),
									},
								},
								},
							},
						},
					},
				}
				swaggerDoc.Paths[method.jsonrpcPath()] = swPath{Post: postMethod}
			} else if service.tags.Contains(tagServerHTTP) && method.tags.Contains(tagMethodHTTP) {
				doc.log.WithField("module", "swagger").Infof("service %s append HTTP method %s", serviceTags, method.Name)
				httpValue, found := swaggerDoc.Paths[method.httpPathSwagger()]
				if !found {
					swaggerDoc.Paths[method.httpPathSwagger()] = swPath{}
				}
				requestContentType := method.tags.Value(tagRequestContentType, contentJSON)
				responseContentType := method.tags.Value(tagResponseContentType, contentJSON)
				httpMethod := &swOperation{
					Summary:     method.tags.Value(tagSummary),
					Description: method.tags.Value(tagDesc),
					Parameters:  parameters,
					Tags:        serviceTags,
					Deprecated:  method.tags.Contains(tagDeprecated),
					RequestBody: &swRequestBody{
						Content: doc.clearContent(swContent{}),
					},
					Responses: swResponses{
						fmt.Sprintf("%d", successCode): swResponse{
							Description: codeToText(successCode),
							Headers:     retHeaders,
							Content: doc.clearContent(swContent{
								responseContentType: swMedia{Schema: swSchema{Ref: "#/components/schemas/" + method.responseStructName()}},
							}),
						},
					},
				}
				if len(method.arguments()) != 0 {
					httpMethod.RequestBody = &swRequestBody{
						Content: doc.clearContent(swContent{
							requestContentType: swMedia{Schema: swSchema{Ref: "#/components/schemas/" + method.requestStructName()}},
						}),
					}
				}
				var methodTags tags.DocTags
				doc.fillErrors(httpMethod.Responses, methodTags.Merge(service.tags).Merge(method.tags))

				if httpMethod.RequestBody.Content == nil {
					httpMethod.RequestBody = nil
				}
				reflect.ValueOf(&httpValue).Elem().FieldByName(utils.ToCamel(strings.ToLower(method.httpMethod()))).Set(reflect.ValueOf(httpMethod))
				swaggerDoc.Paths[method.httpPathSwagger()] = httpValue
			}
		}
	}
	var docData []byte
	swaggerDoc.Components.Schemas = doc.schemas
	if strings.ToLower(filepath.Ext(outFilePath)) == ".json" {
		if docData, err = json.MarshalIndent(swaggerDoc, " ", "    "); err != nil {
			return
		}
	} else {
		if docData, err = yaml.Marshal(swaggerDoc); err != nil {
			return
		}
	}
	doc.log.Info("write to ", outFilePath)
	return os.WriteFile(outFilePath, docData, 0600)
}

func (doc *swagger) fillErrors(responses swResponses, tags tags.DocTags) {

	for key, value := range tags {

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		code, _ := strconv.Atoi(key)

		var content swContent
		var pkgPath, typeName string

		if text, found := statusText[code]; found {

			if value == "skip" {
				continue
			}

			if value != "" {
				if tokens := strings.Split(value, ":"); len(tokens) == 2 {

					pkgPath = tokens[0]
					typeName = tokens[1]

					if retType := doc.searchType(pkgPath, typeName); retType != nil {
						content = swContent{contentJSON: swMedia{Schema: doc.walkVariable(typeName, pkgPath, retType, nil)}}
					}
				}
			}
			responses[key] = swResponse{Description: text, Content: content}

		} else if key == "defaultError" {

			if value != "" {
				if tokens := strings.Split(value, ":"); len(tokens) == 2 {

					pkgPath = tokens[0]
					typeName = tokens[1]

					if retType := doc.searchType(pkgPath, typeName); retType != nil {
						content = swContent{contentJSON: swMedia{Schema: doc.walkVariable(typeName, pkgPath, retType, nil)}}
					}
				}
			}
			responses["default"] = swResponse{Description: "Generic error", Content: content}
		}
	}
}

func (doc *swagger) clearContent(content swContent) swContent {

	if len(content) == 0 {
		return nil
	}
	return content
}
