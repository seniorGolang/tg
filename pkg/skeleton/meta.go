// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (meta.go at 14.05.2020, 2:21) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package skeleton

type metaInfo struct {
	repoName    string
	baseDir     string
	projectName string

	withMongo  bool
	withTracer bool
}
