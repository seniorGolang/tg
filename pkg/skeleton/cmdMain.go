// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (cmdMain.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"

	"github.com/seniorGolang/tg/v2/pkg/astra"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/v2/pkg/logger"
	"github.com/seniorGolang/tg/v2/pkg/tags"
	"github.com/seniorGolang/tg/v2/pkg/utils"
)

var log = logger.Log.WithField("module", "skeleton")

func UpdateCmdMain(repoName, baseDir string, jaeger, mongo bool) (err error) {

	var projectName string
	if projectName, err = getProjectName(path.Join(baseDir, "cmd", "service", "main.go")); err != nil {
		return
	}

	var pkgBase string
	if pkgBase, err = goModRepo(path.Join(baseDir, "go.mod")); err != nil {
		return
	}

	meta := metaInfo{
		baseDir:     baseDir,
		repoName:    repoName,
		projectName: projectName,
		withMongo:   mongo,
	}

	if err = genConfig(meta); err != nil {
		return
	}
	return makeCmdMain(meta, pkgBase, path.Join(meta.baseDir, "cmd", "service"))
}

func makeCmdMain(meta metaInfo, pkgBase, mainPath string) (err error) {

	log.Infof("make %s", path.Join(mainPath, "main.go"))

	serviceDirectory := path.Join(meta.baseDir, "pkg", meta.projectName, "service")

	var files []os.DirEntry
	if files, err = os.ReadDir(serviceDirectory); err != nil {
		return
	}

	var services []types.Interface

	for _, file := range files {

		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		var serviceAst *types.File
		if serviceAst, err = astra.ParseFile(path.Join(serviceDirectory, file.Name())); err != nil {
			return
		}

		for _, iface := range serviceAst.Interfaces {
			if len(tags.ParseTags(iface.Docs)) != 0 {
				services = append(services, iface)
			}
		}
	}
	return renderMain(meta, pkgBase, mainPath, services)
}

func renderMain(meta metaInfo, basePkg, mainPath string, services []types.Interface) (err error) {

	if err = os.MkdirAll(mainPath, os.ModePerm); err != nil {
		return
	}

	srcFile := NewFile("main")

	srcFile.ImportName(pkgLog, "logrus")
	srcFile.ImportName(pkgSignal, "signal")
	srcFile.ImportName(pkgSyscall, "syscall")
	srcFile.ImportName(path.Join(basePkg, "pkg", meta.projectName, "config"), "config")
	srcFile.ImportName(path.Join(basePkg, "pkg", meta.projectName, "service"), "service")
	srcFile.ImportName(path.Join(basePkg, "pkg", meta.projectName, "transport"), "transport")

	srcFile.Add(Const().Id("serviceName").Op("=").Lit(meta.projectName))
	srcFile.Var().Id("log").Op("=").Qual(pkgLog, "New").Call()
	srcFile.Line().Add(renderMainFunc(meta, basePkg, services))

	return srcFile.Save(path.Join(mainPath, "main.go"))
}

func renderMainFunc(meta metaInfo, basePkg string, services []types.Interface) Code {
	return Func().Id("main").Params().BlockFunc(func(g *Group) {
		pkgConfig := path.Join(basePkg, "pkg", meta.projectName, "config")
		g.Line().If(Qual(pkgConfig, "Service").Call().Dot("ReportCaller")).Block(
			Id("log").Dot("SetReportCaller").Call(Lit(true)),
		)
		g.If(List(Id("level"), Err()).Op(":=").Qual(pkgLog, "ParseLevel").Call(Qual(pkgConfig, "Service").Call().Dot("LogLevel")).Op(";").Err().Op("==").Nil()).Block(
			Id("log").Dot("SetLevel").Call(Id("level")),
		)
		g.Line().Id("shutdown").Op(":=").Make(Chan().Qual(pkgOS, "Signal"), Lit(1))
		g.Qual(pkgSignal, "Notify").Call(Id("shutdown"), Qual(pkgSyscall, "SIGINT"))
		g.Line().Defer().Id("log").Dot("Info").Call(Lit("msg"), Lit("goodbye"))
		pkgService := path.Join(basePkg, "pkg", meta.projectName, "service")
		appArgs := []Code{Id("log")}
		pkgTransport := path.Join(basePkg, "pkg", meta.projectName, "transport")
		g.Line().Comment("TODO implement me!")
		for _, service := range services {
			svcName := service.Name
			g.Var().Id(svcName).Qual(pkgService, service.Name)
			appArgs = append(appArgs, Qual(pkgTransport, "Service").Call(Qual(pkgTransport, "New"+svcName).Call(Id("log"), Id(svcName))))
		}
		g.Line()
		if meta.withTracer {
			g.Id("srv").Op(":=").Qual(pkgTransport, "New").Call(appArgs...).Dot("WithTrace").Call()
		} else {
			g.Id("srv").Op(":=").Qual(pkgTransport, "New").Call(appArgs...)
		}
		g.Line()
		serve := make(map[string]struct{})
		for _, service := range services {
			if tags.ParseTags(service.Docs).IsSet("http-server") {
				serve["http-server"] = struct{}{}
			}
			if tags.ParseTags(service.Docs).IsSet("jsonRPC-server") {
				serve["http-server"] = struct{}{}
			}
			if tags.ParseTags(service.Docs).IsSet("test") {
				serve["test"] = struct{}{}
			}
		}
		if _, found := serve["http-server"]; found {
			g.Id("srv").Dot("ServeHTTP").Call(Qual(pkgConfig, "Service").Call().Dot("ServiceBind"))
		}
		g.Line().Qual(pkgTransport, "ServeMetrics").Call(Id("log"), Qual(pkgConfig, "Service").Call().Dot("MetricsBind"))
		g.Line().Op("<-").Id("shutdown")
		g.Line().Id("log").Dot("Info").Call(Lit("shutdown application"))
		g.Id("srv").Dot("Shutdown").Call()
	})
}

func getProjectName(mainPath string) (projectName string, err error) {

	var fileAst *ast.File
	fSet := token.NewFileSet()
	parserMode := parser.ParseComments

	if fileAst, err = parser.ParseFile(fSet, mainPath, nil, parserMode); err != nil {
		return
	}

	for _, d := range fileAst.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, id := range spec.Names {
						if id.Name == "serviceName" {
							projectName = id.Obj.Decl.(*ast.ValueSpec).Values[0].(*ast.BasicLit).Value
							projectName = strings.TrimPrefix(projectName, "\"")
							projectName = strings.TrimSuffix(projectName, "\"")
							return
						}
					}
				}
			}
		}
	}
	err = errors.New("constant 'serviceName not found'")
	return
}

func goModRepo(goModPath string) (repo string, err error) {
	return utils.GetPkgPath(goModPath, false)
}
