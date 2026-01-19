// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package executor

import (
	"github.com/seniorGolang/tg/v3/internal/installer/models"
)

func detectKind(installation *models.Installation) (kind Kind) {

	if installation.Kind != "" {
		return Kind(installation.Kind)
	}

	if len(installation.Commands) > 0 {
		return KindCommand
	}

	return KindStage
}
