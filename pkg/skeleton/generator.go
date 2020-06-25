// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (generator.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/pkg/generator"
)

func GenerateSkeleton(log logrus.FieldLogger, projectName, repoName, baseDir string, jaeger, zipkin, mongo bool) (err error) {

	meta := metaInfo{
		baseDir:     baseDir,
		repoName:    repoName,
		projectName: projectName,
		withMongo:   mongo,
	}

	if jaeger {
		meta.tracer = TracerJaeger
	}
	if zipkin {
		meta.tracer = TracerZipkin
	}

	log.Info("init go.mod")

	packageName := meta.repoName

	if packageName == "" {
		packageName = path.Join(meta.repoName, meta.projectName)
	}

	if err = exec.Command("go", "mod", "init", path.Join(meta.repoName)).Run(); err != nil {
		log.Warning("go.mod already exist")
	}

	if err = genConfig(meta); err != nil {
		return
	}

	if err = genServices(meta); err != nil {
		return
	}

	var tr generator.Transport
	if tr, err = generator.NewTransport(log, path.Join(meta.baseDir, "pkg", projectName, "service")); err == nil {
		if err = tr.RenderServer(path.Join(meta.baseDir, "pkg", projectName)); err != nil {
			return
		}
	} else {
		return
	}

	if err = makeCmdMain(meta, meta.repoName, path.Join(meta.baseDir, "cmd", projectName)); err != nil {
		return
	}

	log.Info("download dependencies ...")
	return exec.Command("go", "mod", "tidy").Run()
}
