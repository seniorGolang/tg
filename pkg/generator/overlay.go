// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (overlay.go) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (tr *Transport) jsonRPCOverlayKeys() (headerNames []string, cookieNames []string) {

	headers := make(map[string]struct{})
	cookies := make(map[string]struct{})
	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		if !svc.isJsonRPC() {
			continue
		}
		for _, method := range svc.methods {
			if !method.isJsonRPC() {
				continue
			}
			for argName, key := range method.varHeaderMap() {
				if method.argByName(argName) == nil || key == "" {
					continue
				}
				headers[strings.TrimPrefix(key, "!")] = struct{}{}
			}
			for argName, key := range method.varCookieMap() {
				if method.argByName(argName) == nil || key == "" {
					continue
				}
				cookies[strings.TrimPrefix(key, "!")] = struct{}{}
			}
		}
	}
	return sortedMapKeys(headers), sortedMapKeys(cookies)
}

func (tr *Transport) jsonRPCNeedsOverlay() (ok bool) {

	headerNames, cookieNames := tr.jsonRPCOverlayKeys()
	return len(headerNames)+len(cookieNames) > 0
}

func (svc *service) jsonRPCNeedsOverlay() (ok bool) {

	for _, method := range svc.methods {
		if method.isJsonRPC() && method.hasFiberRequest() {
			return true
		}
	}
	return false
}

func overlayKeyToFieldName(key string) (name string) {

	parts := strings.Split(key, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}

func (tr *Transport) requestOverlayKeyType() (c Code) {

	return Line().
		Type().Id("requestOverlayKey").Struct().Line().
		Var().Id("keyRequestOverlay").Op("=").Id("requestOverlayKey").Values()
}

func (tr *Transport) requestOverlayStructType() (c Code) {

	headerNames, cookieNames := tr.jsonRPCOverlayKeys()
	fields := make(map[string]struct{}, len(headerNames)+len(cookieNames))
	return Type().Id("requestOverlay").StructFunc(func(tg *Group) {
		for _, name := range headerNames {
			field := overlayKeyToFieldName(name)
			if _, exists := fields[field]; exists {
				continue
			}
			fields[field] = struct{}{}
			tg.Id(field).String()
		}
		for _, name := range cookieNames {
			field := overlayKeyToFieldName(name)
			if _, exists := fields[field]; exists {
				continue
			}
			fields[field] = struct{}{}
			tg.Id(field).String()
		}
	})
}

func (tr *Transport) requestOverlayGetMethod() (c Code) {

	headerNames, cookieNames := tr.jsonRPCOverlayKeys()
	seen := make(map[string]struct{}, len(headerNames)+len(cookieNames))
	cases := make([]Code, 0, len(headerNames)+len(cookieNames)+1)
	for _, name := range headerNames {
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		cases = append(cases, Case(Lit(name)).Block(Return(Id("o").Dot(overlayKeyToFieldName(name)))))
	}
	for _, name := range cookieNames {
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		cases = append(cases, Case(Lit(name)).Block(Return(Id("o").Dot(overlayKeyToFieldName(name)))))
	}
	cases = append(cases, Default().Block(Return(Lit(""))))
	return Func().Params(Id("o").Id("requestOverlay")).Id("Get").Params(Id("key").String()).Params(String()).Block(
		Switch(Id("key")).Block(cases...),
	)
}

func (tr *Transport) requestOverlayFromFiberFunc() (c Code) {

	headerNames, cookieNames := tr.jsonRPCOverlayKeys()
	return Line().Func().Id("requestOverlayFromFiber").
		Params(Id(_ctx_).Op("*").Qual(packageFiber, "Ctx")).
		Params(Id("overlay").Id("requestOverlay")).
		BlockFunc(func(block *Group) {
			for _, name := range headerNames {
				block.Id("overlay").Dot(overlayKeyToFieldName(name)).Op("=").
					Qual(packageStrings, "Clone").Call(String().Call(Id(_ctx_).Dot("Request").Call().Dot("Header").Dot("Peek").Call(Lit(name))))
			}
			for _, name := range cookieNames {
				block.Id("overlay").Dot(overlayKeyToFieldName(name)).Op("=").
					Qual(packageStrings, "Clone").Call(Id(_ctx_).Dot("Cookies").Call(Lit(name)))
			}
			block.Return()
		})
}

func setRequestOverlayContext() (c Code) {

	return Id(_ctx_).Dot("SetUserContext").Call(
		Qual(packageContext, "WithValue").Call(
			Id(_ctx_).Dot("UserContext").Call(),
			Id("keyRequestOverlay"),
			Id("requestOverlayFromFiber").Call(Id(_ctx_)),
		),
	)
}

func (m *method) applyOverlayFromContext(errStatement func(arg, header string) *Statement) (block *Statement) {

	headerMap := m.varHeaderMap()
	cookieMap := m.varCookieMap()
	hasArgs := false
	for argName := range headerMap {
		if m.argByName(argName) != nil {
			hasArgs = true
			break
		}
	}
	if !hasArgs {
		for argName := range cookieMap {
			if m.argByName(argName) != nil {
				hasArgs = true
				break
			}
		}
	}
	if !hasArgs {
		return Line()
	}
	inner := Line()
	inner.Add(m.argFromString("header", headerMap,
		func(srcName string) Code {
			srcName = strings.TrimPrefix(srcName, "!")
			return Id("overlay").Dot("Get").Call(Lit(srcName))
		},
		errStatement,
	))
	inner.Add(m.argFromString("cookie", cookieMap,
		func(srcName string) Code {
			srcName = strings.TrimPrefix(srcName, "!")
			return Id("overlay").Dot("Get").Call(Lit(srcName))
		},
		errStatement,
	))
	return Line().If(List(Id("overlay"), Id("ok")).Op(":=").Id("userCtx").Dot("Value").Call(Id("keyRequestOverlay")).Assert(Id("requestOverlay")).Op(";").Id("ok")).Block(inner)
}
