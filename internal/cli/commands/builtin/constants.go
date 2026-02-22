// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

const (
	cmdNameTG            = "tg"
	pathSeparator        = "/"
	versionPrefixV       = "v"
	cmdSubPluginDoc      = "doc"
	cmdPathPluginDoc     = "plugin"
	versionSeparator     = "@"
	commandPathSeparator = " "

	zshrcFile            = ".zshrc"
	shCommand            = "sh"
	shellZsh             = "zsh"
	shellBash            = "bash"
	shellFish            = "fish"
	bashrcFile           = ".bashrc"
	homeEnvVar           = "HOME"
	shellEnvVar          = "SHELL"
	zshFuncPrefix        = "_"
	shEchoCommand        = "echo $0"
	shCommandFlag        = "-c"
	powershellExe        = "powershell.exe"
	fishConfigFile       = ".config/fish/config.fish"
	bashProfileFile      = ".bash_profile"
	zshFunctionsDir      = ".zsh/functions"
	shellPowershell      = "powershell"
	bashCompletionDir    = ".bash_completion.d"
	fishCompletionsDir   = ".config/fish/completions"
	fishCompletionSuffix = ".fish"

	filePermDir  = 0700
	filePermFile = 0600

	goModFile           = "go.mod"
	gitlabCIFile        = ".gitlab-ci.yml"
	deployTypeNone      = "none"
	deployTypeGitlab    = "gitlab"
	deployTypeGithub    = "github"
	githubWorkflowsDir  = ".github"
	githubWorkflowsFile = "workflows/deploy.yml"

	docVersionSeparator     = " v"
	docDescriptionSeparator = " - "

	zshFpathConfig          = "\nfpath=(~/.zsh/functions $fpath)\nautoload -U compinit && compinit\n"
	fishSourcePrefix        = "source ~/"
	messageTruncateStart    = 47
	messageTruncateSuffix   = "..."
	messageTruncateLength   = 50
	completionSourceSuffix  = "\n"
	completionSourcePrefix  = "\nsource "
	completionCommentPrefix = "\n# Completion for "

	paramIndent                     = "  "
	paramSeparator                  = ": "
	errorSeparator                  = ": "
	pluginNamePrefix                = "["
	pluginNameSuffix                = "] "
	tableRowSeparator               = "-"
	progressBarTotal                = 100
	packageSourceSuffix             = ")"
	packageSourcePrefix             = " ("
	packageFoundInMultipleManifests = "Package "
	foundInMultipleManifestsSuffix  = " found in multiple manifests"

	optionKeyOut               = "out"
	optionKeyDir               = "dir"
	optionKeyKind              = "kind"
	optionKeyName              = "name"
	optionKeyForce             = "force"
	optionKeyClean             = "clean"
	optionKeyDryRun            = "dry-run"
	optionKeyCommand           = "command"
	optionKeyLicense           = "license"
	optionKeyVersion           = "version"
	optionKeyVerbose           = "verbose"
	optionKeyNoCascade         = "no-cascade"
	optionKeyModuleName        = "module-name"
	optionKeyDeployType        = "deploy-type"
	optionKeyOverrideManifest  = "override-manifest"
	optionKeySkipVersionUpdate = "skip-version-update"
)
