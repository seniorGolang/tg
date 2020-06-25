// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (cmdMain.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/vetcher/go-astra"
	"github.com/vetcher/go-astra/types"

	. "github.com/dave/jennifer/jen"

	"github.com/seniorGolang/tg/pkg/logger"
	"github.com/seniorGolang/tg/pkg/tags"
	"github.com/seniorGolang/tg/pkg/utils"
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
		tracer:      TracerJaeger,
		withMongo:   mongo,
	}

	if err = genConfig(meta); err != nil {
		return
	}

	if !jaeger {
		meta.tracer = TracerZipkin
	}
	return makeCmdMain(meta, pkgBase, path.Join(meta.baseDir, "cmd", "service"))
}

func makeCmdMain(meta metaInfo, pkgBase, mainPath string) (err error) {

	log.Infof("make %s", path.Join(mainPath, "main.go"))

	serviceDirectory := path.Join(meta.baseDir, "pkg", meta.projectName, "service")

	var files []os.FileInfo
	if files, err = ioutil.ReadDir(serviceDirectory); err != nil {
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

	srcFile.ImportName(pkgLog, "logger")
	srcFile.ImportName(pkgUtils, "utils")
	srcFile.ImportName(pkgMongo, "mongo")
	srcFile.ImportName(pkgSignal, "signal")
	srcFile.ImportName(pkgSyscall, "syscall")
	srcFile.ImportName(path.Join(basePkg, "pkg", meta.projectName, "config"), "config")
	srcFile.ImportName(path.Join(basePkg, "pkg", meta.projectName, "service"), "service")
	srcFile.ImportName(path.Join(basePkg, "pkg", meta.projectName, "transport", "server"), "server")

	srcFile.Add(renderMainVars())
	srcFile.Line()
	srcFile.Add(renderMainConst(meta))
	srcFile.Line().Var().Id("log").Op("=").Qual(pkgLog, "Log").Dot("WithField").Call(Lit("module"), Id("serviceName")).Line()
	srcFile.Add(renderMainFunc(meta, basePkg, services))

	return srcFile.Save(path.Join(mainPath, "main.go"))
}

func renderMainFunc(meta metaInfo, basePkg string, services []types.Interface) Code {

	return Func().Id("main").Params().BlockFunc(func(g *Group) {

		g.Line()

		pkgConfig := path.Join(basePkg, "pkg", meta.projectName, "config")

		g.Qual(pkgConfig, "SetBuildInfo").Call(Id("serviceName"), Id("GitSHA"), Id("Version"), Id("BuildStamp"), Id("BuildNumber"))

		g.Line()

		g.Id("shutdown").Op(":=").Make(Chan().Qual(pkgOS, "Signal"), Lit(1))
		g.Qual(pkgSignal, "Notify").Call(Id("shutdown"), Qual(pkgSyscall, "SIGINT"))

		g.Line()

		g.Defer().Id("log").Dot("Info").Call(Lit("msg"), Lit("goodbye"))

		g.Line()

		if meta.withMongo {
			g.Id("mongoDB").Op(":=").Qual(pkgMongo, "Connect").Call(Qual(pkgConfig, "Values").Call().Dot("MongoAddr"))
			g.Defer().Id("mongoDB").Dot("ConnSession").Call().Dot("Close").Call()
		}

		g.Line()

		pkgService := path.Join(basePkg, "pkg", meta.projectName, "service")

		appArgs := []Code{Qual(pkgConfig, "App").Call()}

		g.Comment("TODO implement me!")

		for _, service := range services {
			svcName := "svc" + utils.ToCamel(service.Name)
			g.Var().Id(svcName).Qual(pkgService, service.Name)
			appArgs = append(appArgs, Id(svcName))
		}

		g.Line()

		pkgServer := path.Join(basePkg, "pkg", meta.projectName, "transport", "server")

		if meta.tracer == TracerJaeger {
			g.Id("app").Op(":=").Qual(pkgServer, "New").Call(appArgs...).Dot("WithJaeger").Call()
		} else if meta.tracer == TracerZipkin {
			g.Id("app").Op(":=").Qual(pkgServer, "New").Call(appArgs...).Dot("WithZipkin").Call(Qual(pkgConfig, "Values").Call().Dot("Zipkin"))
		} else {
			g.Id("app").Op(":=").Qual(pkgServer, "New").Call(appArgs...)
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
			g.Id("app").Dot("ServeHTTP").Call()
		}

		g.Id("app").Dot("ServePPROF").Call()
		g.Id("app").Dot("ServeMetrics").Call()

		g.Line()

		g.Op("<-").Id("shutdown")

		g.Line()

		g.Id("log").Dot("Info").Call(Lit("shutdown application"))
		g.Id("app").Dot("Shutdown").Call()
	})
}

func renderMainVars() Code {
	return Var().Op("(").Line().
		Id("GitSHA").Op("=").Lit("").Line().
		Id("Version").Op("=").Lit("").Line().
		Id("BuildStamp").Op("=").Lit("").Line().
		Id("BuildNumber").Op("=").Lit("").Line().
		Op(")")
}

func renderMainConst(meta metaInfo) Code {
	return Const().Id("serviceName").Op("=").Lit(meta.projectName)
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
