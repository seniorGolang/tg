// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

type CICDCreator struct{}

func (c *CICDCreator) Create(rootDir string, deployType string) (err error) {

	data := TemplateData{}

	var templatePath string
	var outputPath string

	switch deployType {
	case DeployTypeGitLab:
		templatePath = "templates/cicd_gitlab.tmpl"
		// В WASM файловая система монтируется в корень "/", поэтому используем относительные пути
		outputPath = GitLabCIFileName
	case DeployTypeGitHub:
		templatePath = "templates/cicd_github.tmpl"
		outputPath = filepath.Join(GitHubWorkflowsDir, GitHubWorkflowsSubDir, GitHubDeployFileName)
	default:
		return
	}

	var content string
	if content, err = renderTemplate(templatePath, data); err != nil {
		return fmt.Errorf(i18n.Msg("failed to render CI/CD template: %w"), err)
	}

	if err = writeFile(outputPath, content); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to create %s: %w"), "CI/CD configuration", err)
	}

	return
}

func (c *CICDCreator) DetectDeployType(rootDir string) (deployType string) {

	gitlabPath := GitLabCIFileName
	var gitlabStatErr error
	if _, gitlabStatErr = os.Stat(gitlabPath); gitlabStatErr == nil {
		deployType = DeployTypeGitLab
		return
	}

	githubPath := filepath.Join(GitHubWorkflowsDir, GitHubWorkflowsSubDir, GitHubDeployFileName)
	var githubStatErr error
	if _, githubStatErr = os.Stat(githubPath); githubStatErr == nil {
		deployType = DeployTypeGitHub
		return
	}

	return DeployTypeNone
}
