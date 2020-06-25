// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (configApp.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"path"

	. "github.com/dave/jennifer/jen"
)

func renderConfigApp(meta metaInfo, configPath string) (err error) {

	srcFile := NewFile("config")

	srcFile.Add(renderConfigAppType())
	srcFile.Add(renderConfigAppInstance())
	srcFile.Add(renderConfigAppMethods())

	return srcFile.Save(path.Join(configPath, "application.go"))
}

func renderConfigAppType() Code {
	return Type().Id("configApplication").Struct()
}

func renderConfigAppInstance() Code {
	return Func().Id("App").Params().Op("*").Id("configApplication").Block(
		Return(Op("&").Id("configApplication").Op("{}")),
	)
}

func renderConfigAppMethods() (code *Statement) {

	code = &Statement{}

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("NodeName").Params().String().Block(
		Return(Id("NodeName").Call()),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("ServiceName").Params().String().Block(
		Return(Id("ServiceName").Call()),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("Version").Params().String().Block(
		Return(Id("Version").Call()),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("GitSHA").Params().String().Block(
		Return(Id("GitSHA").Call()),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("BuildStamp").Params().String().Block(
		Return(Id("BuildStamp").Call()),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("BuildNumber").Params().String().Block(
		Return(Id("BuildNumber").Call()),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("BindAddr").Params().String().Block(
		Return(Id("Values").Call().Dot("ServiceBind")),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("EnablePPROF").Params().Bool().Block(
		Return(Id("Values").Call().Dot("EnablePPROF")),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("BindPPROF").Params().String().Block(
		Return(Id("Values").Call().Dot("PprofBind")),
	).Line()

	code.Line().Func().Params(Id("c").Op("*").Id("configApplication")).Id("MetricsAddr").Params().String().Block(
		Return(Id("Values").Call().Dot("MetricsBind")),
	).Line()
	return
}
