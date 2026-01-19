// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package env

// GetValue: для Go-переменных — go env -json, для остальных — os.Getenv.
func GetValue(key string) (value string, ok bool) {

	if key == "" {
		return "", false
	}

	goProvider := newGoEnvProvider()
	osProvider := newOSEnvProvider()

	if isGoEnvVar(key) {
		return goProvider.get(key)
	}
	return osProvider.get(key)
}

func ResolveEnvVars(allowedEnvVars []string) (envVars map[string]string) {

	envVars = resolve(allowedEnvVars)
	return
}

func resolve(allowedEnvVars []string) (envVars map[string]string) {

	envVars = make(map[string]string)

	if len(allowedEnvVars) == 0 {
		return
	}

	goProvider := newGoEnvProvider()
	osProvider := newOSEnvProvider()

	for _, key := range allowedEnvVars {
		var value string
		var ok bool

		if isGoEnvVar(key) {
			value, ok = goProvider.get(key)
		} else {
			value, ok = osProvider.get(key)
		}

		if ok {
			envVars[key] = value
		}
	}

	return
}
