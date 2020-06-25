// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (options.go at 16.05.2020, 13:47) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

type Option func(svc *service)

func WithTests(path string) Option {
	return func(svc *service) {
		svc.testsPath = path
	}
}

func WithImplements(path string) Option {
	return func(svc *service) {
		svc.implementsPath = path
	}
}
