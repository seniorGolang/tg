// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package plugin

type Plugin interface {
	Info() (info Info)

	Execute(rootDir string, request Storage, path ...string) (response Storage, err error)
}
