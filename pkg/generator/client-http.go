// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-jsonrpc.go at 25.06.2020, 10:50) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

func (tr *Transport) renderClientHTTP(outDir string) error {

	if err := pkgCopyTo("cb", outDir); err != nil {
		return err
	}
	if err := pkgCopyTo("hasher", outDir); err != nil {
		return err
	}
	// Initialize a new source file
	srcFile := newSrc(filepath.Base(outDir))
	srcFile.PackageComment(doNotEdit)
	// Import necessary packages
	srcFile.ImportName("context", "context")
	srcFile.ImportName("net/http", "http")
	srcFile.ImportName("encoding/json", "json")
	srcFile.ImportName("os", "os")
	srcFile.ImportName("time", "time")
	srcFile.ImportName("github.com/gofiber/fiber/v2", "fiber")
	// Import internal packages
	srcFile.ImportName(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "cb")
	srcFile.ImportName(fmt.Sprintf("%s/hasher", tr.pkgPath(outDir)), "hasher")
	// srcFile.ImportName(fmt.Sprintf("%s/httpclient", tr.pkgPath(outDir)), "httpclient")
	// Generate the ClientHTTP struct
	srcFile.Line().Add(tr.httpClientStructFunc(outDir))
	// Save the generated source file
	return srcFile.Save(path.Join(outDir, "client-http.go"))
}

func (tr *Transport) httpClientStructFunc(outDir string) Code {

	return Type().Id("ClientHTTP").Struct(
		Id("name").String(),
		Id("httpClient").Op("*").Qual("github.com/gofiber/fiber/v2", "Client"),
		Line(),
		Id("cache").Id("cache"),
		Line(),
		Id("cbCfg").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "Settings"),
		Id("cb").Op("*").Qual(fmt.Sprintf("%s/cb", tr.pkgPath(outDir)), "CircuitBreaker"),
		Line(),
		Id("fallbackTTL").Qual("time", "Duration"),
		Line(),
		Id("options").Id("clientOptions"),
	)
}
