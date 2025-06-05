// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (generator.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

//go:embed templates
var templates embed.FS

func GenerateSkeleton(log logrus.FieldLogger, moduleName, projectName, serviceName, baseDir string) (err error) {

	if baseDir, err = filepath.Abs(baseDir); err != nil {
		return
	}
	if err = os.MkdirAll(baseDir, 0777); err != nil {
		return
	}
	if err = os.Chdir(baseDir); err != nil {
		return
	}
	meta := map[string]string{
		"moduleName":       moduleName,
		"projectName":      projectName,
		"projectNameCamel": utils.ToCamel(projectName),
		"serviceNameCamel": utils.ToCamel(serviceName),
		"serviceName":      utils.ToLowerCamel(serviceName),
	}
	log.Info("init go.mod")
	if err = exec.Command("go", "mod", "init", moduleName).Run(); err != nil {
		log.WithError(err).Warning("go mod creation error")
		return
	}
	var tmpl *template.Template
	if tmpl, err = template.ParseFS(templates, "templates/*.tmpl"); err != nil {
		log.WithError(err).Warning("template parse error")
		return
	}
	log.Info("make main.go")
	if err = renderFile(tmpl, "main.tmpl", path.Join(baseDir, "cmd", serviceName, "main.go"), meta); err != nil {
		log.WithError(err).Warning("render main.go error")
		return
	}
	log.Info("make config")
	if err = renderFile(tmpl, "config.tmpl", path.Join(baseDir, "internal", "config", "service.go"), meta); err != nil {
		log.WithError(err).Warning("render service.go error")
		return
	}
	log.Info("make utils")
	if err = renderFile(tmpl, "headers.tmpl", path.Join(baseDir, "internal", "utils", "header", "headers.go"), meta); err != nil {
		log.WithError(err).Warning("render headers.go error")
		return
	}
	if err = renderFile(tmpl, "golangci-lint.tmpl", path.Join(baseDir, ".golangci.yml"), meta); err != nil {
		log.WithError(err).Warning("render .golangci.yml error")
		return
	}
	if err = renderFile(tmpl, "ignore.tmpl", path.Join(baseDir, ".gitignore"), meta); err != nil {
		log.WithError(err).Warning("render .gitignore error")
		return
	}
	if err = renderFile(tmpl, "health.tmpl", path.Join(baseDir, "internal", "utils", "health.go"), meta); err != nil {
		log.WithError(err).Warning("render health.go error")
		return
	}
	log.Info("make contracts")
	if err = renderFile(tmpl, "interface.tmpl", path.Join(baseDir, "contracts", fmt.Sprintf("%s.go", utils.ToLowerCamel(serviceName))), meta); err != nil {
		log.WithError(err).Warning("render contracts error")
		return
	}
	if err = pkgCopyTo("dto", path.Join(baseDir, "contracts")); err != nil {
		return err
	}
	if err = renderFile(tmpl, "tg.tmpl", path.Join(baseDir, "contracts", "tg.go"), meta); err != nil {
		log.WithError(err).Warning("render tg.go error")
		return
	}
	if err = os.MkdirAll(path.Join(baseDir, "contracts", "dto"), 0777); err != nil {
		log.WithError(err).Warning("make types dir error")
		return
	}
	log.Info("make services")
	if err = renderFile(tmpl, "service.tmpl", path.Join(baseDir, "internal", "services", utils.ToLowerCamel(serviceName), "service.go"), meta); err != nil {
		log.WithError(err).Warning("render service.go error")
		return
	}
	if err = renderFile(tmpl, "service_method.tmpl", path.Join(baseDir, "internal", "services", utils.ToLowerCamel(serviceName), "some.go"), meta); err != nil {
		log.WithError(err).Warning("render some.go error")
		return
	}
	log.Info("make errors")
	if err = pkgCopyTo("errors", path.Join(baseDir, "pkg")); err != nil {
		return err
	}
	_ = os.Chdir(path.Join(baseDir, "contracts"))
	if err = exec.Command("go", "generate").Run(); err != nil {
		log.WithError(err).Warning("tg generate error")
		return
	}
	log.Info("download dependencies ...")
	_ = os.Chdir(path.Join(baseDir))
	return exec.Command("go", "mod", "tidy").Run()
}

func renderFile(tmpl *template.Template, template, path string, data any) (err error) {

	_ = os.Remove(path)
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0777); err != nil {
		return
	}
	var buf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buf, template, data); err != nil {
		return
	}
	return os.WriteFile(path, buf.Bytes(), 0600)
}
