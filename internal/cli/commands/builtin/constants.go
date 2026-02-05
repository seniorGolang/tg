// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

const (
	cmdNameTG            = "tg"
	cmdPathPluginDoc     = "plugin"
	cmdSubPluginDoc      = "doc"
	versionPrefixV       = "v"
	versionSeparator     = "@"
	pathSeparator        = "/"
	commandPathSeparator = " "

	homeEnvVar           = "HOME"
	shellEnvVar          = "SHELL"
	bashCompletionDir    = ".bash_completion.d"
	zshFunctionsDir      = ".zsh/functions"
	fishCompletionsDir   = ".config/fish/completions"
	bashrcFile           = ".bashrc"
	bashProfileFile      = ".bash_profile"
	zshrcFile            = ".zshrc"
	fishConfigFile       = ".config/fish/config.fish"
	shellBash            = "bash"
	shellZsh             = "zsh"
	shellFish            = "fish"
	shellPowershell      = "powershell"
	powershellExe        = "powershell.exe"
	shCommand            = "sh"
	shCommandFlag        = "-c"
	shEchoCommand        = "echo $0"
	zshFuncPrefix        = "_"
	fishCompletionSuffix = ".fish"
	filePermDir          = 0700
	filePermFile         = 0600

	goModFile           = "go.mod"
	gitlabCIFile        = ".gitlab-ci.yml"
	githubWorkflowsDir  = ".github"
	githubWorkflowsFile = "workflows/deploy.yml"
	deployTypeGitlab    = "gitlab"
	deployTypeGithub    = "github"
	deployTypeNone      = "none"

	docVersionSeparator     = " v"
	docDescriptionSeparator = " - "

	completionCommentPrefix = "\n# Completion for "
	completionSourcePrefix  = "\nsource "
	completionSourceSuffix  = "\n"
	zshFpathConfig          = "\nfpath=(~/.zsh/functions $fpath)\nautoload -U compinit && compinit\n"
	fishSourcePrefix        = "source ~/"
	messageTruncateLength   = 50
	messageTruncateSuffix   = "..."
	messageTruncateStart    = 47

	packageFoundInMultipleManifests = "Package "
	foundInMultipleManifestsSuffix  = " found in multiple manifests"
	paramIndent                     = "  "
	paramSeparator                  = ": "
	pluginNamePrefix                = "["
	pluginNameSuffix                = "] "
	packageSourcePrefix             = " ("
	packageSourceSuffix             = ")"
	tableRowSeparator               = "-"
	errorSeparator                  = ": "
	progressBarTotal                = 100

	optionKeyVersion    = "version"
	optionKeyForce      = "force"
	optionKeyDryRun     = "dry-run"
	optionKeyVerbose    = "verbose"
	optionKeyNoCascade  = "no-cascade"
	optionKeyName       = "name"
	optionKeyCommand    = "command"
	optionKeyDir        = "dir"
	optionKeyLicense    = "license"
	optionKeyModuleName = "module-name"
	optionKeyDeployType = "deploy-type"
	optionKeyKind       = "kind"
)
