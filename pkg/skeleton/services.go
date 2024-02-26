// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (services.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

import (
	"fmt"
	"os"
	"path"

	. "github.com/dave/jennifer/jen"

	"github.com/DivPro/tg/v2/pkg/utils"
)

func genServices(meta metaInfo) (err error) {

	log.Info("generate services")

	if err = renderBaseService(meta, path.Join(meta.baseDir, "pkg", meta.projectName, "service")); err != nil {
		return
	}

	typesPath := path.Join(meta.baseDir, "pkg", meta.projectName, "service", "types")

	if err = os.MkdirAll(typesPath, os.ModePerm); err != nil {
		return
	}
	return
}

func renderBaseService(meta metaInfo, servicesPath string) (err error) {

	if err = os.MkdirAll(servicesPath, os.ModePerm); err != nil {
		return
	}

	srcFile := NewFile("service")
	srcFile.PackageComment("@tg version=0.0.1")
	srcFile.PackageComment(fmt.Sprintf("@tg backend=%s", meta.projectName))
	srcFile.PackageComment(fmt.Sprintf("@tg title=`%s API`", meta.projectName))
	srcFile.PackageComment(fmt.Sprintf("@tg description=`A service which provide %s API`", meta.projectName))
	srcFile.PackageComment(fmt.Sprintf("@tg servers=`http://%s:9000`", meta.projectName))

	srcFile.ImportName(pkgContext, "context")

	srcFile.Comment("@tg jsonRPC-server log trace metrics test")
	srcFile.Type().Id(utils.ToCamel(meta.projectName)).Interface(
		Id("Method").Params(Id("ctx").Qual(pkgContext, "Context")).Params(Err().Error()),
	)

	return srcFile.Save(path.Join(servicesPath, fmt.Sprintf("%s.go", meta.projectName)))
}
