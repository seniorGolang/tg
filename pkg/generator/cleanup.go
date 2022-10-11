// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (cleanup.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"bufio"
	"os"
	"path"
	"strings"
)

func (tr *Transport) cleanup(outDir string) {

	var err error
	var files []os.DirEntry
	if files, err = os.ReadDir(outDir); err != nil {
		tr.log.WithError(err).Warn("cleanup")
		return
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		filePath := path.Join(outDir, file.Name())
		if goFile, err := os.Open(filePath); err == nil {
			if firstLine, err := bufio.NewReader(goFile).ReadString('\n'); err == nil {
				if strings.TrimSpace(strings.TrimPrefix(firstLine, "//")) == doNotEdit {
					if err = os.Remove(filePath); err != nil {
						tr.log.WithError(err).Warn("cleanup")
					}
				}
			}
			_ = goFile.Close()
		}
	}
}
