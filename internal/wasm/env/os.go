// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package env

import "os"

// osEnvProvider реализует envProvider для обычных переменных окружения.
type osEnvProvider struct{}

func newOSEnvProvider() (provider envProvider) {
	return &osEnvProvider{}
}

func (p *osEnvProvider) get(key string) (value string, ok bool) {

	value = os.Getenv(key)
	if value == "" {
		return "", false
	}

	return value, true
}
