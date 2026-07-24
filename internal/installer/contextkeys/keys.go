// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package contextkeys

type Key string

const (
	Force               Key = "force"
	Source              Key = "source"
	Skipped             Key = "skipped"
	TreeCollector       Key = "treeCollector"
	SkipDatabaseUpdate  Key = "skipDatabaseUpdate"
	SessionInstalledIDs Key = "sessionInstalledIDs"
	Skills              Key = "skills"
)
