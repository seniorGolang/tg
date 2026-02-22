// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"strings"
)

var optionalValueFlagNames = make(map[string]struct{})

func addOptionalValueFlag(name string) {

	optionalValueFlagNames[name] = struct{}{}
}

func PreprocessArgs(args []string) (out []string) {

	out = make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 2 && arg[:2] == "--" && !strings.Contains(arg, "=") {
			flagName := arg[2:]
			if _, ok := optionalValueFlagNames[flagName]; ok {
				nextMissing := i+1 >= len(args)
				nextIsFlag := i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] == '-' && args[i+1] != "--"
				if nextMissing || nextIsFlag {
					out = append(out, "--"+flagName+"=")
					continue
				}
			}
		}
		out = append(out, arg)
	}
	return
}
