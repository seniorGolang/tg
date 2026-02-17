// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package env

import (
	"os/exec"
	"sync"

	"github.com/goccy/go-json"
)

var (
	goEnvVars   map[string]string
	goEnvVarsMu sync.Once
)

// initGoEnvVars инициализирует список Go-переменных окружения через go env -json.
func initGoEnvVars() {

	goEnvVarsMu.Do(func() {
		goEnvVars = make(map[string]string)

		cmd := exec.Command("go", "env", "-json")
		var err error
		var output []byte
		if output, err = cmd.Output(); err != nil {
			return
		}

		var envMap map[string]string
		if err = json.Unmarshal(output, &envMap); err != nil {
			return
		}

		for key, value := range envMap {
			goEnvVars[key] = value
		}
	})
}

func isGoEnvVar(key string) (isGo bool) {

	initGoEnvVars()
	_, isGo = goEnvVars[key]
	return
}

// goEnvProvider реализует envProvider для Go-переменных окружения.
type goEnvProvider struct{}

func newGoEnvProvider() (provider envProvider) {
	return &goEnvProvider{}
}

func (p *goEnvProvider) get(key string) (value string, ok bool) {

	initGoEnvVars()
	value, ok = goEnvVars[key]
	if !ok || value == "" {
		return "", false
	}
	return
}
