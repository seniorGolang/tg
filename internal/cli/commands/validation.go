// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

func hasRequiredOptions(options []Option) (hasRequired bool) {

	for _, opt := range options {
		if opt.Required {
			hasRequired = true
			return
		}
	}
	return
}

func validateRequiredOptions(options map[string]any, commandOptions []Option) (allProvided bool) {

	allProvided = true
	for _, opt := range commandOptions {
		if opt.Required {
			val, exists := options[opt.Name]
			if !exists || val == "" || val == nil {
				return false
			}
		}
	}
	return
}

func getRequiredOptions(commandOptions []Option) (requiredOpts []string) {

	requiredOpts = make([]string, 0)
	for _, opt := range commandOptions {
		if opt.Required {
			requiredOpts = append(requiredOpts, opt.Name)
		}
	}
	return
}
