// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (config.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"os"
	"path"

	. "github.com/dave/jennifer/jen"
)

func genConfig(meta metaInfo) (err error) {

	log.Info("generate config")

	configPath := path.Join(meta.baseDir, "pkg", meta.projectName, "config")

	if err = os.MkdirAll(configPath, os.ModePerm); err != nil {
		return
	}
	if err = renderServiceConfig(configPath); err != nil {
		return
	}
	if meta.withMongo {
		if err = renderMongoConfig(meta, configPath); err != nil {
			return
		}
	}
	return
}

func renderServiceConfig(configPath string) (err error) {

	srcFile := NewFile("config")

	srcFile.ImportName(pkgEnv, "envconfig")

	srcFile.Line().Type().Id("ServiceConfig").StructFunc(func(g *Group) {
		g.Id("LogLevel").String().Tag(map[string]string{"envconfig": "LOG_LEVEL", "default": "debug"})
		g.Id("ReportCaller").Bool().Tag(map[string]string{"envconfig": "LOG_REPORT_CALLER", "default": "false"})
		g.Id("ServiceBind").String().Tag(map[string]string{"envconfig": "BIND_ADDR", "default": ":9000"})
		g.Id("HealthBind").String().Tag(map[string]string{"envconfig": "BIND_HEALTH", "default": ":9091"})
		g.Id("MetricsBind").String().Tag(map[string]string{"envconfig": "BIND_METRICS", "default": ":9090"})
	})

	srcFile.Line().Var().Id("service").Op("*").Id("ServiceConfig")

	srcFile.Line().Func().Id("Service").Params().Id("ServiceConfig").Block(
		Line(),
		If(Id("service").Op("!=").Nil()).Block(
			Return(Op("*").Id("service")),
		),
		Id("service").Op("=").Op("&").Id("ServiceConfig").Op("{}"),
		If(Err().Op(":=").Qual(pkgEnv, "Process").Call(Lit(""), Id("service")).Op(";").Err().Op("!=").Nil()).Block(
			Panic(Err()),
		),
		Return(Op("*").Id("service")),
	)
	return srcFile.Save(path.Join(configPath, "service.go"))
}

func renderMongoConfig(meta metaInfo, configPath string) (err error) {

	srcFile := NewFile("config")

	srcFile.ImportName(pkgEnv, "envconfig")

	srcFile.Line().Type().Id("MongoConfig").StructFunc(func(g *Group) {
		g.Id("Address").String().Tag(map[string]string{"envconfig": "MONGO_ADDR", "default": "mongodb://localhost/" + meta.projectName})
	})

	srcFile.Line().Var().Id("mongo").Op("*").Id("MongoConfig")

	srcFile.Line().Func().Id("Mongo").Params().Id("MongoConfig").Block(
		Line(),
		If(Id("mongo").Op("!=").Nil()).Block(
			Return(Op("*").Id("mongo")),
		),
		Id("mongo").Op("=").Op("&").Id("MongoConfig").Op("{}"),
		If(Err().Op(":=").Qual(pkgEnv, "Process").Call(Lit(""), Id("mongo")).Op(";").Err().Op("!=").Nil()).Block(
			Panic(Err()),
		),
		Return(Op("*").Id("mongo")),
	)
	return srcFile.Save(path.Join(configPath, "mongo.go"))
}
