// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

const (
	cmdNameTG            = "tg"
	cmdSubPluginDoc      = "doc"
	pathSeparator        = "/"
	cmdPathPluginDoc     = "plugin"
	versionPrefixV       = "v"
	versionSeparator     = "@"
	commandPathSeparator = " "

	homeEnvVar           = "HOME"
	shellEnvVar          = "SHELL"
	zshFuncPrefix        = "_"
	shCommand            = "sh"
	shCommandFlag        = "-c"
	shellZsh             = "zsh"
	shellFish            = "fish"
	bashrcFile           = ".bashrc"
	shellBash            = "bash"
	zshrcFile            = ".zshrc"
	shEchoCommand        = "echo $0"
	bashProfileFile      = ".bash_profile"
	filePermDir          = 0700
	filePermFile         = 0600
	powershellExe        = "powershell.exe"
	shellPowershell      = "powershell"
	bashCompletionDir    = ".bash_completion.d"
	zshFunctionsDir      = ".zsh/functions"
	fishCompletionSuffix = ".fish"
	fishCompletionsDir   = ".config/fish/completions"
	fishConfigFile       = ".config/fish/config.fish"

	goModFile           = "go.mod"
	deployTypeNone      = "none"
	githubWorkflowsDir  = ".github"
	deployTypeGithub    = "github"
	deployTypeGitlab    = "gitlab"
	gitlabCIFile        = ".gitlab-ci.yml"
	githubWorkflowsFile = "workflows/deploy.yml"

	docVersionSeparator     = " v"
	docDescriptionSeparator = " - "

	completionSourceSuffix  = "\n"
	fishSourcePrefix        = "source ~/"
	messageTruncateSuffix   = "..."
	completionSourcePrefix  = "\nsource "
	messageTruncateLength   = 50
	messageTruncateStart    = 47
	completionCommentPrefix = "\n# Completion for "
	zshFpathConfig          = "\nfpath=(~/.zsh/functions $fpath)\nautoload -U compinit && compinit\n"

	paramIndent                     = "  "
	paramSeparator                  = ": "
	tableRowSeparator               = "-"
	errorSeparator                  = ": "
	pluginNamePrefix                = "["
	pluginNameSuffix                = "] "
	packageSourcePrefix             = " ("
	packageSourceSuffix             = ")"
	progressBarTotal                = 100
	packageFoundInMultipleManifests = "Package "
	foundInMultipleManifestsSuffix  = " found in multiple manifests"

	optionKeyDir               = "dir"
	optionKeyKind              = "kind"
	optionKeyName              = "name"
	optionKeyOut               = "out"
	optionKeyForce             = "force"
	optionKeyClean             = "clean"
	optionKeyCommand           = "command"
	optionKeyLicense           = "license"
	optionKeyVerbose           = "verbose"
	optionKeyVersion           = "version"
	optionKeyDryRun            = "dry-run"
	optionKeyNoCascade         = "no-cascade"
	optionKeyDeployType        = "deploy-type"
	optionKeyModuleName        = "module-name"
	optionKeyOverrideManifest  = "override-manifest"
	optionKeySkipVersionUpdate = "skip-version-update"
)
