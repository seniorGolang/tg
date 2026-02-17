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

	commandStatusKey     = "_command_status_"
	commandStatusSuccess = "success"
	commandStatusError   = "error"

	executionPlanKey = "_execute_plan_"

	cyclePathSeparator = " -> "

	optionKeyCommandPath = "_command_path"
)
