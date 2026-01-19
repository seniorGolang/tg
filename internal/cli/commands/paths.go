// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

// Константы путей команд
const (
	// Команда воспроизведения сохраненных плагинов
	cmdPathReplay = "replay"

	// Команды управления пакетами
	cmdPathPkgAdd     = "pkg"
	cmdPathPkgList    = "pkg"
	cmdPathPkgUpgrade = "pkg"
	cmdPathPkgUpdate  = "pkg"
	cmdPathPkgDel     = "pkg"
	cmdPathPkgInfo    = "pkg"
	cmdPathPkgRepo    = "pkg"
	cmdSubPkgAdd      = "add"
	cmdSubPkgList     = "list"
	cmdSubPkgUpgrade  = "upgrade"
	cmdSubPkgUpdate   = "update"
	cmdSubPkgDel      = "del"
	cmdSubPkgInfo     = "info"
	cmdSubPkgRepo     = "repo"

	// Команды разработки плагинов
	cmdPathPluginDoc    = "plugin"
	cmdPathPluginUpdate = "plugin"
	cmdPathPluginInit   = "plugin"
	cmdPathPluginAdd    = "plugin"
	cmdSubPluginDoc     = "doc"
	cmdSubPluginUpdate  = "update"
	cmdSubPluginInit    = "init"
	cmdSubPluginAdd     = "add"

	// Группа команд управления scope
	cmdPathPkgScope   = "pkg"
	cmdSubPkgScope    = "scope"
	cmdSubScopeCreate = "create"
	cmdSubScopeUse    = "use"
	cmdSubScopeList   = "list"
	cmdSubScopeDelete = "delete"
	cmdSubScopeShow   = "show"

	// Группа команд completion
	cmdGroupCompletion         = "completion"
	cmdSubCompletionBash       = "bash"
	cmdSubCompletionZsh        = "zsh"
	cmdSubCompletionFish       = "fish"
	cmdSubCompletionPowershell = "powershell"
	cmdSubCompletionInstall    = "install"

	commandPathSeparator = " "
	cmdNameTG            = "tg"
	pathAliasSeparator   = ":"
)
