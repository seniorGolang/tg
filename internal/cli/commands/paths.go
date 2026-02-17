// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

// Константы путей команд
const (
	// Команда воспроизведения сохраненных плагинов
	cmdPathReplay = "replay"

	// Команды управления пакетами
	cmdSubPkgAdd      = "add"
	cmdSubPkgDel      = "del"
	cmdSubPkgInfo     = "info"
	cmdSubPkgList     = "list"
	cmdSubPkgRepo     = "repo"
	cmdSubPkgUpdate   = "update"
	cmdSubPkgUpgrade  = "upgrade"
	cmdPathPkgAdd     = "pkg"
	cmdPathPkgDel     = "pkg"
	cmdPathPkgInfo    = "pkg"
	cmdPathPkgList    = "pkg"
	cmdPathPkgRepo    = "pkg"
	cmdPathPkgUpdate  = "pkg"
	cmdPathPkgUpgrade = "pkg"

	// Команды разработки плагинов
	cmdSubPluginAdd     = "add"
	cmdSubPluginBuild   = "build"
	cmdSubPluginDoc     = "doc"
	cmdSubPluginInit    = "init"
	cmdSubPluginUpdate  = "update"
	cmdPathPluginAdd    = "plugin"
	cmdPathPluginBuild  = "plugin"
	cmdPathPluginDoc    = "plugin"
	cmdPathPluginInit   = "plugin"
	cmdPathPluginUpdate = "plugin"

	// Группа команд управления scope
	cmdSubScopeDel  = "del"
	cmdSubScopeList = "list"
	cmdSubScopeShow = "show"
	cmdSubScopeUse  = "use"
	cmdPathPkgScope = "pkg"
	cmdSubPkgScope  = "scope"

	// Группа команд completion
	cmdSubCompletionBash       = "bash"
	cmdSubCompletionFish       = "fish"
	cmdSubCompletionZsh        = "zsh"
	cmdSubCompletionInstall    = "install"
	cmdSubCompletionPowershell = "powershell"
	cmdGroupCompletion         = "completion"

	cmdNameTG            = "tg"
	pathAliasSeparator   = ":"
	commandPathSeparator = " "
)
