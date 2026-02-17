// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package builtin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/cli/types"
	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type CompletionGenerator interface {
	GetName() (name string)
	GenZshCompletion(writer any) (err error)
	GenPowerShellCompletion(writer any) (err error)
	GenFishCompletion(writer any, includeDesc bool) (err error)
	GenBashCompletionV2(writer any, includeDesc bool) (err error)
}

var globalCompletionGenerator CompletionGenerator

func SetCompletionGenerator(generator CompletionGenerator) {

	globalCompletionGenerator = generator
}

func HandleCompletionBash(ctx types.CommandContext) (err error) {

	if globalCompletionGenerator == nil {
		return errors.New(i18n.Msg("Completion generator not set"))
	}

	var buf bytes.Buffer
	if err = globalCompletionGenerator.GenBashCompletionV2(&buf, true); err != nil {
		errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "bash") + err.Error()
		return errors.New(errorMsg)
	}

	fmt.Print(buf.String())
	return
}

func HandleCompletionZsh(ctx types.CommandContext) (err error) {

	if globalCompletionGenerator == nil {
		return errors.New(i18n.Msg("Completion generator not set"))
	}

	var buf bytes.Buffer
	if err = globalCompletionGenerator.GenZshCompletion(&buf); err != nil {
		errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "zsh") + err.Error()
		return errors.New(errorMsg)
	}

	fmt.Print(buf.String())
	return
}

func HandleCompletionFish(ctx types.CommandContext) (err error) {

	if globalCompletionGenerator == nil {
		return errors.New(i18n.Msg("Completion generator not set"))
	}

	var buf bytes.Buffer
	if err = globalCompletionGenerator.GenFishCompletion(&buf, true); err != nil {
		errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "fish") + err.Error()
		return errors.New(errorMsg)
	}

	fmt.Print(buf.String())
	return
}

func HandleCompletionPowershell(ctx types.CommandContext) (err error) {

	if globalCompletionGenerator == nil {
		return errors.New(i18n.Msg("Completion generator not set"))
	}

	var buf bytes.Buffer
	if err = globalCompletionGenerator.GenPowerShellCompletion(&buf); err != nil {
		errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "powershell") + err.Error()
		return errors.New(errorMsg)
	}

	fmt.Print(buf.String())
	return
}

func HandleCompletionInstall(ctx types.CommandContext) (err error) {

	if globalCompletionGenerator == nil {
		return errors.New(i18n.Msg("Completion generator not set"))
	}

	shell := detectShell()
	if shell == "" {
		return errors.New(i18n.Msg("Failed to detect shell. Specify shell explicitly via --shell option"))
	}

	ctx.Logger.Info(i18n.Msg("Shell detected"), "shell", shell)

	var sourceLine string
	var completionFile string
	var scriptBuf bytes.Buffer
	cmdName := globalCompletionGenerator.GetName()

	switch shell {
	case shellBash:
		if err = globalCompletionGenerator.GenBashCompletionV2(&scriptBuf, true); err != nil {
			errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "bash") + err.Error()
			return errors.New(errorMsg)
		}
		completionFile = filepath.Join(os.Getenv(homeEnvVar), bashCompletionDir, cmdName)
		sourceLine = completionCommentPrefix + cmdName + completionSourcePrefix + completionFile + completionSourceSuffix
	case shellZsh:
		if err = globalCompletionGenerator.GenZshCompletion(&scriptBuf); err != nil {
			errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "zsh") + err.Error()
			return errors.New(errorMsg)
		}
		zshFuncDir := filepath.Join(os.Getenv(homeEnvVar), zshFunctionsDir)
		completionFile = filepath.Join(zshFuncDir, zshFuncPrefix+cmdName)
		sourceLine = completionCommentPrefix + cmdName + zshFpathConfig
	case shellFish:
		if err = globalCompletionGenerator.GenFishCompletion(&scriptBuf, true); err != nil {
			errorMsg := fmt.Sprintf(i18n.Msg("Error generating %s completion: "), "fish") + err.Error()
			return errors.New(errorMsg)
		}
		fishCompDir := filepath.Join(os.Getenv(homeEnvVar), fishCompletionsDir)
		completionFile = filepath.Join(fishCompDir, cmdName+fishCompletionSuffix)
		sourceLine = ""
	default:
		unsupportedMsg := i18n.Msg("Unsupported shell: ") + shell
		return errors.New(unsupportedMsg)
	}

	completionDir := filepath.Dir(completionFile)
	if err = os.MkdirAll(completionDir, filePermDir); err != nil {
		errorMsg := i18n.Msg("Error creating directory ") + completionDir + errorSeparator + err.Error()
		return errors.New(errorMsg)
	}

	if err = os.WriteFile(completionFile, scriptBuf.Bytes(), filePermFile); err != nil {
		errorMsg := i18n.Msg("Error writing completion file: ") + err.Error()
		return errors.New(errorMsg)
	}

	ctx.Logger.Info(i18n.Msg("Completion script installed"), "file", completionFile)

	var configFile string
	switch shell {
	case shellBash:
		configFile = findBashConfigFile()
	case shellZsh:
		configFile = filepath.Join(os.Getenv(homeEnvVar), zshrcFile)
	case shellFish:
		fishConfigPath := fishSourcePrefix + fishConfigFile
		restartMsg := i18n.Msg("Completion installed. Restart shell or run: ") + fishConfigPath
		ctx.Logger.Info(restartMsg)
		return
	}

	if configFile != "" {
		var configContent []byte
		if configContent, err = os.ReadFile(configFile); err != nil && !os.IsNotExist(err) {
			ctx.Logger.Warn(i18n.Msg("Failed to read config file"), "file", configFile, "error", err)
		} else {
			if strings.Contains(string(configContent), completionFile) {
				ctx.Logger.Info(i18n.Msg("Completion already configured in config file"), "file", configFile)
			} else {
				var file *os.File
				if file, err = os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePermFile); err != nil {
					ctx.Logger.Warn(i18n.Msg("Failed to open config file for writing"), "file", configFile, "error", err)
					ctx.Logger.Info(i18n.Msg("Add the following line to your shell config file:"), "line", sourceLine)
				} else {
					defer file.Close()
					if _, err = file.WriteString(sourceLine); err != nil {
						ctx.Logger.Warn(i18n.Msg("Failed to write to config file"), "file", configFile, "error", err)
						ctx.Logger.Info(i18n.Msg("Add the following line to your shell config file:"), "line", sourceLine)
					} else {
						ctx.Logger.Info(i18n.Msg("Config file updated"), "file", configFile)
					}
				}
			}
		}
	}

	ctx.Logger.Info(i18n.Msg("Completion successfully installed!"), "shell", shell)
	restartMsg := i18n.Msg("Restart shell or run: source ") + configFile
	ctx.Logger.Info(restartMsg)

	return
}

func findBashConfigFile() (configFile string) {

	homeDir := os.Getenv(homeEnvVar)
	bashrcPath := filepath.Join(homeDir, bashrcFile)
	var bashrcErr error
	if _, bashrcErr = os.Stat(bashrcPath); bashrcErr == nil {
		return bashrcPath
	}

	bashProfilePath := filepath.Join(homeDir, bashProfileFile)
	var bashProfileErr error
	if _, bashProfileErr = os.Stat(bashProfilePath); bashProfileErr == nil {
		return bashProfilePath
	}

	return bashrcPath
}

func detectShell() (shell string) {

	shellEnv := os.Getenv(shellEnvVar)
	if shellEnv != "" {
		shellName := filepath.Base(shellEnv)
		if strings.Contains(shellName, shellBash) {
			return shellBash
		}
		if strings.Contains(shellName, shellZsh) {
			return shellZsh
		}
		if strings.Contains(shellName, shellFish) {
			return shellFish
		}
	}

	if runtime.GOOS == "windows" {
		var lookPathErr error
		if _, lookPathErr = exec.LookPath(powershellExe); lookPathErr == nil {
			return shellPowershell
		}
		return ""
	}

	cmd := exec.Command(shCommand, shCommandFlag, shEchoCommand)
	var err error
	var output []byte
	if output, err = cmd.Output(); err == nil {
		shellName := strings.TrimSpace(string(output))
		if strings.Contains(shellName, shellBash) {
			return shellBash
		}
		if strings.Contains(shellName, shellZsh) {
			return shellZsh
		}
		if strings.Contains(shellName, shellFish) {
			return shellFish
		}
	}

	return ""
}
