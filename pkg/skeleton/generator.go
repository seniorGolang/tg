// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (generator.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/DivPro/tg/v2/pkg/generator"
)

func GenerateSkeleton(log logrus.FieldLogger, version, projectName, repoName, baseDir string, trace, mongo bool) (err error) {

	if baseDir, err = filepath.Abs(baseDir); err != nil {
		return
	}
	if err = os.MkdirAll(baseDir, 0777); err != nil {
		return
	}
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	meta := metaInfo{
		baseDir:     baseDir,
		repoName:    repoName,
		projectName: projectName,
		withTracer:  trace,
		withMongo:   mongo,
	}

	log.Info("init go.mod")

	packageName := meta.repoName

	if packageName == "" {
		packageName = path.Join(meta.repoName, meta.projectName)
	}

	cmdMakeMod := exec.Command("go", "mod", "init", packageName)
	if err = cmdMakeMod.Run(); err != nil {
		log.WithError(err).Warning("go mod creation error")
	}

	if err = genConfig(meta); err != nil {
		return
	}

	if err = genServices(meta); err != nil {
		return
	}

	var tr generator.Transport
	if tr, err = generator.NewTransport(log, version, path.Join(meta.baseDir, "pkg", projectName, "service")); err == nil {
		if err = tr.RenderServer(path.Join(meta.baseDir, "pkg", projectName, "transport")); err != nil {
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
