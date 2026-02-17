// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package models

// Version представляет структуру версии.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Build      string
	Original   string
}

// VersionConstraint представляет ограничение версии.
type VersionConstraint struct {
	Operator string
	Version  *Version
}
