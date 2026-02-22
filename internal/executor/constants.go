// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

const (
	versionPrefixV                = "v"
	requirementPrefixCaret        = "^"
	requirementPrefixTilde        = "~"
	requirementPrefixGreaterEqual = ">="
	requirementPrefixLessEqual    = "<="
	requirementPrefixGreater      = ">"
	requirementPrefixLess         = "<"
	requirementPrefixEqual        = "="

	commandStatusError   = "_error_"
	commandStatusSuccess = "_success_"
	commandStatusKey     = "_command_status_"

	executionPlanKey = "_execute_plan_"

	cyclePathSeparator = " -> "

	optionKeyRunDir      = "_runDir_"
	optionKeyCommandPath = "_command_path_"
)
