// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

import (
	"net/url"
	"strings"
)

type DependencyGraph struct {
	Nodes map[string]*DependencyNode
	Edges []*DependencyEdge
}

type DependencyNode struct {
	Package *Package
	Version Version
	ID      string
}

type DependencyEdge struct {
	From       *DependencyNode
	To         *DependencyNode
	Dependency *Dependency
}

// ParseDependencyString: форматы package, package@version, source:package, source:package@version, URL:package@version.
func ParseDependencyString(depStr string) (dep Dependency) {

	depStr = strings.TrimSpace(depStr)
	if depStr == "" {
		return
	}

	parts := strings.Split(depStr, "@")
	specWithoutVersion := parts[0]
	if len(parts) == 2 {
		dep.Version = parts[1]
	}

	colonIndex := strings.LastIndex(specWithoutVersion, ":")
	if colonIndex > 0 {
		schemeEndIndex := strings.Index(specWithoutVersion, "://")
		if schemeEndIndex < 0 || colonIndex > schemeEndIndex+2 {
			beforeColon := specWithoutVersion[:colonIndex]
			afterColon := specWithoutVersion[colonIndex+1:]

			testURL := beforeColon
			if !strings.Contains(testURL, "://") {
				testURL = "https://" + testURL
			}
			testParsedURL, testErr := url.Parse(testURL)
			if testErr == nil && testParsedURL.Scheme != "" {
				if !strings.Contains(beforeColon, "://") {
					dep.Source = "https://" + beforeColon
				} else {
					dep.Source = beforeColon
				}
				dep.Package = afterColon
				return
			}
		}
	}

	dep.Package = specWithoutVersion
	return
}
