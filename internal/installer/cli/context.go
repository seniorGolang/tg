// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import "github.com/seniorGolang/tg/v3/internal/installer/contextkeys"

// ContextKeyForce - ключ для передачи флага force через контекст.
var ContextKeyForce = contextkeys.Force

// ContextKeySkipped - ключ для передачи флага skipped через контекст.
var ContextKeySkipped = contextkeys.Skipped

// ContextKeySource - ключ для передачи source через контекст.
var ContextKeySource = contextkeys.Source
