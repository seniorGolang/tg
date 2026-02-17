// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package commands

import (
	"errors"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/cli/commands/builtin"
	"github.com/seniorGolang/tg/v3/internal/cli/plugin/generator"
	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// BuiltinCommand представляет встроенную команду
type BuiltinCommand struct {
	path        []string
	options     []Option
	executor    func(ctx CommandContext) (err error)
	description string
}

func (c *BuiltinCommand) GetPath() (path []string) {

	if c == nil {
		return nil
	}
	return c.path
}

func (c *BuiltinCommand) GetDescription() (description string) {

	if c == nil {
		return ""
	}
	return c.description
}

func (c *BuiltinCommand) GetOptions() (options []Option) {

	if c == nil {
		return nil
	}
	return c.options
}

func (c *BuiltinCommand) Execute(ctx CommandContext) (err error) {

	if c.executor == nil {
		return errors.New(i18n.Msg("builtin command executor is not set"))
	}
	return c.executor(ctx)
}

func isBuiltinCommand(cmd Command) (isBuiltin bool) {

	_, isBuiltin = cmd.(*BuiltinCommand)
	return
}

// registerBuiltinCommands регистрирует все встроенные команды
func registerBuiltinCommands(tree *CommandTree) (err error) {

	// Команда replay
	replayCmd := &BuiltinCommand{
		path:        []string{cmdPathReplay},
		description: i18n.Msg("Replay saved plugin commands"),
		options:     []Option{},
		executor: func(ctx CommandContext) (err error) {
			return builtin.HandleUpdateWithPrompt(ctx, func(opts []types.Option, current map[string]any, path []string) map[string]any {
				cmdOpts := make([]Option, len(opts))
				copy(cmdOpts, opts)
				return PromptCommandOptionsFromPlugin(cmdOpts, current, path)
			})
		},
	}
	if err = tree.RegisterCommand(replayCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "replay", err)
	}

	// Команда plugin doc
	pluginDocCmd := &BuiltinCommand{
		path:        []string{cmdPathPluginDoc, cmdSubPluginDoc},
		description: i18n.Msg("Show plugin documentation"),
		options: []Option{
			{
				Name:         "plugin",
				Type:         optionTypeString,
				Required:     false,
				Description:  i18n.Msg("Plugin name (version can be specified via @)"),
				IsPositional: true,
			},
		},
		executor: builtin.HandleDoc,
	}
	if err = tree.RegisterCommand(pluginDocCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "plugin doc", err)
	}

	// Команда plugin init
	pluginInitCmd := &BuiltinCommand{
		path:        []string{cmdPathPluginInit, cmdSubPluginInit},
		description: i18n.Msg("Create template for new plugin"),
		options: []Option{
			{Name: "name", Type: optionTypeString, Short: "n", Required: true, Description: i18n.Msg("Plugin name")},
			{Name: "command", Type: optionTypeString, Short: "c", Required: false, Description: i18n.Msg("CLI command name")},
			{Name: "deploy-type", Type: optionTypeString, Short: "t", Required: false, Description: i18n.Msg("Deploy type: gitlab, github, none")},
			{Name: "license", Type: optionTypeString, Short: "l", Required: false, Description: i18n.Msg("Plugin license"), Default: generator.DefaultLicense},
			{Name: "module-name", Type: optionTypeString, Short: "m", Required: false, Description: i18n.Msg("Module name for shared code"), Default: generator.DefaultModuleName},
			{Name: "kind", Type: optionTypeString, Short: "k", Required: false, Description: i18n.Msg("Plugin kind: pre, stage, command, post")},
		},
		executor: builtin.HandlePluginInit,
	}
	if err = tree.RegisterCommand(pluginInitCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "plugin init", err)
	}

	// Команда plugin add
	pluginAddCmd := &BuiltinCommand{
		path:        []string{cmdPathPluginAdd, cmdSubPluginAdd},
		description: i18n.Msg("Add new plugin to existing repository"),
		options: []Option{
			{Name: "name", Type: optionTypeString, Short: "n", Required: true, Description: i18n.Msg("Plugin name")},
			{Name: "command", Type: optionTypeString, Short: "c", Required: false, Description: i18n.Msg("CLI command name (optional for transformer plugins)")},
			{Name: "dir", Type: optionTypeString, Short: "d", Required: false, Description: i18n.Msg("Plugin directory (default: plugins/{name})")},
			{Name: "license", Type: optionTypeString, Short: "l", Required: false, Description: i18n.Msg("Plugin license"), Default: generator.DefaultLicense},
			{Name: "module-name", Type: optionTypeString, Short: "m", Required: false, Description: i18n.Msg("Module name for shared code"), Default: generator.DefaultModuleName},
			{Name: "kind", Type: optionTypeString, Short: "k", Required: false, Description: i18n.Msg("Plugin kind: pre, stage, command, post")},
		},
		executor: builtin.HandlePluginAdd,
	}
	if err = tree.RegisterCommand(pluginAddCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "plugin add", err)
	}

	// Команда plugin build
	pluginBuildCmd := &BuiltinCommand{
		path:        []string{cmdPathPluginBuild, cmdSubPluginBuild},
		description: i18n.Msg("Build plugins and manifest"),
		options: []Option{
			{Name: "out", Type: optionTypeString, Description: i18n.Msg("Output directory"), Required: false, Default: "./dist"},
			{Name: "clean", Type: optionTypeBool, Description: i18n.Msg("Clean output directory before build"), Required: false},
			{Name: "override-manifest", Type: optionTypeString, Description: i18n.Msg("Path to manifest overrides file"), Required: false, Default: "./manifest.overrides.yml"},
			{Name: "version", Type: optionTypeString, Description: i18n.Msg("Manifest version (default from git tag)"), Required: false},
			{Name: "skip-version-update", Type: optionTypeBool, Description: i18n.Msg("Do not write internal/version.go"), Required: false},
		},
		executor: builtin.HandlePluginBuild,
	}
	if err = tree.RegisterCommand(pluginBuildCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "plugin build", err)
	}

	// Команда pkg add
	pkgAddCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgAdd, cmdSubPkgAdd},
		description: i18n.Msg("Install package"),
		options: []Option{
			{
				Name:         "package",
				Type:         optionTypeString,
				Description:  i18n.Msg("Package name (version can be specified via @)"),
				Required:     true,
				IsPositional: true,
			},
			{
				Name:        "version",
				Type:        optionTypeString,
				Description: i18n.Msg("Specify version"),
				Required:    false,
			},
			{
				Name:        "force",
				Type:        optionTypeBool,
				Description: i18n.Msg("Force installation (overwrite existing)"),
				Required:    false,
			},
			{
				Name:        "dry-run",
				Type:        optionTypeBool,
				Description: i18n.Msg("Simulate installation without actual actions"),
				Required:    false,
			},
			{
				Name:        "verbose",
				Short:       "v",
				Type:        optionTypeBool,
				Description: i18n.Msg("Verbose output"),
				Required:    false,
			},
		},
		executor: builtin.HandlePluginInstall,
	}
	if err = tree.RegisterCommand(pkgAddCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg add", err)
	}

	// Команда pkg del
	pkgDelCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgDel, cmdSubPkgDel},
		description: i18n.Msg("Remove package"),
		options: []Option{
			{
				Name:         "package",
				Type:         optionTypeString,
				Description:  i18n.Msg("Package name to remove"),
				Required:     true,
				IsPositional: true,
			},
			{
				Name:        "no-cascade",
				Type:        optionTypeBool,
				Description: i18n.Msg("Disable cascade removal of dependencies (by default dependencies are removed if they are not used elsewhere)"),
				Required:    false,
			},
			{
				Name:        "dry-run",
				Type:        optionTypeBool,
				Description: i18n.Msg("Simulate removal"),
				Required:    false,
			},
		},
		executor: builtin.HandlePluginRemove,
	}
	if err = tree.RegisterCommand(pkgDelCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg del", err)
	}

	// Команда pkg list
	pkgListCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgList, cmdSubPkgList},
		description: i18n.Msg("List installed packages"),
		options:     []Option{},
		executor:    builtin.HandlePluginList,
	}
	if err = tree.RegisterCommand(pkgListCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg list", err)
	}

	// Команда pkg repo
	pkgRepoCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgRepo, cmdSubPkgRepo},
		description: i18n.Msg("Add manifest to catalog"),
		options: []Option{
			{
				Name:         "manifest-url",
				Type:         optionTypeString,
				Description:  i18n.Msg("Manifest URL"),
				Required:     true,
				IsPositional: true,
			},
			{
				Name:        "force",
				Type:        optionTypeBool,
				Description: i18n.Msg("Force reload (even if already loaded)"),
				Required:    false,
			},
			{
				Name:        "verbose",
				Type:        optionTypeBool,
				Description: i18n.Msg("Verbose output of cascade loading process"),
				Required:    false,
			},
		},
		executor: builtin.HandlePluginRepo,
	}
	if err = tree.RegisterCommand(pkgRepoCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg repo", err)
	}

	// Команда pkg update
	pkgUpdateCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgUpdate, cmdSubPkgUpdate},
		description: i18n.Msg("Update manifests"),
		options: []Option{
			{
				Name:        "force",
				Type:        optionTypeBool,
				Description: i18n.Msg("Force update (ignore change check)"),
				Required:    false,
			},
			{
				Name:        "verbose",
				Type:        optionTypeBool,
				Description: i18n.Msg("Verbose output"),
				Required:    false,
			},
		},
		executor: builtin.HandlePluginUpdate,
	}
	if err = tree.RegisterCommand(pkgUpdateCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg update", err)
	}

	// Команда plugin update
	pluginUpdateCmd := &BuiltinCommand{
		path:        []string{cmdPathPluginUpdate, cmdSubPluginUpdate},
		description: i18n.Msg("Update installed packages"),
		options:     []Option{},
		executor:    builtin.HandlePluginUpgrade,
	}
	if err = tree.RegisterCommand(pluginUpdateCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "plugin update", err)
	}

	// Команда pkg info
	pkgInfoCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgInfo, cmdSubPkgInfo},
		description: i18n.Msg("Package information"),
		options: []Option{
			{
				Name:         "package",
				Type:         optionTypeString,
				Description:  i18n.Msg("Package name"),
				Required:     false,
				IsPositional: true,
			},
		},
		executor: builtin.HandlePluginInfo,
	}
	if err = tree.RegisterCommand(pkgInfoCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg info", err)
	}

	// Команда pkg scope use
	pkgScopeUseCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgScope, cmdSubPkgScope, cmdSubScopeUse},
		description: i18n.Msg("Switch to specified scope"),
		options: []Option{
			{
				Name:         "name",
				Type:         optionTypeString,
				Description:  i18n.Msg("Scope name"),
				Required:     true,
				IsPositional: true,
			},
		},
		executor: builtin.HandlePluginScopeUse,
	}
	if err = tree.RegisterCommand(pkgScopeUseCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg scope use", err)
	}

	// Команда pkg scope list
	pkgScopeListCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgScope, cmdSubPkgScope, cmdSubScopeList},
		description: i18n.Msg("List all scopes"),
		options:     []Option{},
		executor:    builtin.HandlePluginScopeList,
	}
	if err = tree.RegisterCommand(pkgScopeListCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg scope list", err)
	}

	// Команда pkg scope del
	pkgScopeDelCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgScope, cmdSubPkgScope, cmdSubScopeDel},
		description: i18n.Msg("Delete scope"),
		options: []Option{
			{
				Name:         "name",
				Type:         optionTypeString,
				Description:  i18n.Msg("Scope name"),
				Required:     true,
				IsPositional: true,
			},
			{
				Name:        "force",
				Type:        optionTypeBool,
				Description: i18n.Msg("Force deletion"),
				Required:    false,
			},
		},
		executor: builtin.HandlePluginScopeDelete,
	}
	if err = tree.RegisterCommand(pkgScopeDelCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg scope del", err)
	}

	// Команда pkg scope show
	pkgScopeShowCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgScope, cmdSubPkgScope, cmdSubScopeShow},
		description: i18n.Msg("Show scope information"),
		options: []Option{
			{
				Name:         "name",
				Type:         optionTypeString,
				Description:  i18n.Msg("Scope name (optional)"),
				Required:     false,
				IsPositional: true,
			},
		},
		executor: builtin.HandlePluginScopeShow,
	}
	if err = tree.RegisterCommand(pkgScopeShowCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg scope show", err)
	}

	// Команда pkg upgrade
	pkgUpgradeCmd := &BuiltinCommand{
		path:        []string{cmdPathPkgUpgrade, cmdSubPkgUpgrade},
		description: i18n.Msg("Update installed packages"),
		options: []Option{
			{
				Name:         "package",
				Type:         optionTypeString,
				Description:  i18n.Msg("Package name (optional)"),
				Required:     false,
				IsPositional: true,
			},
		},
		executor: builtin.HandlePluginUpgrade,
	}
	if err = tree.RegisterCommand(pkgUpgradeCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "pkg upgrade", err)
	}

	// Команда completion bash
	completionBashCmd := &BuiltinCommand{
		path:        []string{cmdGroupCompletion, cmdSubCompletionBash},
		description: i18n.Msg("Generate bash completion script"),
		options:     []Option{},
		executor:    builtin.HandleCompletionBash,
	}
	if err = tree.RegisterCommand(completionBashCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "completion bash", err)
	}

	// Команда completion zsh
	completionZshCmd := &BuiltinCommand{
		path:        []string{cmdGroupCompletion, cmdSubCompletionZsh},
		description: i18n.Msg("Generate zsh completion script"),
		options:     []Option{},
		executor:    builtin.HandleCompletionZsh,
	}
	if err = tree.RegisterCommand(completionZshCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "completion zsh", err)
	}

	// Команда completion fish
	completionFishCmd := &BuiltinCommand{
		path:        []string{cmdGroupCompletion, cmdSubCompletionFish},
		description: i18n.Msg("Generate fish completion script"),
		options:     []Option{},
		executor:    builtin.HandleCompletionFish,
	}
	if err = tree.RegisterCommand(completionFishCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "completion fish", err)
	}

	// Команда completion powershell
	completionPowershellCmd := &BuiltinCommand{
		path:        []string{cmdGroupCompletion, cmdSubCompletionPowershell},
		description: i18n.Msg("Generate PowerShell completion script"),
		options:     []Option{},
		executor:    builtin.HandleCompletionPowershell,
	}
	if err = tree.RegisterCommand(completionPowershellCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "completion powershell", err)
	}

	// Команда completion install
	completionInstallCmd := &BuiltinCommand{
		path:        []string{cmdGroupCompletion, cmdSubCompletionInstall},
		description: i18n.Msg("Automatically install completion for current shell"),
		options:     []Option{},
		executor:    builtin.HandleCompletionInstall,
	}
	if err = tree.RegisterCommand(completionInstallCmd); err != nil {
		return fmt.Errorf(i18n.Msg("Error registering command %s")+": %w", "completion install", err)
	}

	return
}
