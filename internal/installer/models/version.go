// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Build      string
	Original   string
}

type VersionConstraint struct {
	Operator string
	Version  *Version
}
