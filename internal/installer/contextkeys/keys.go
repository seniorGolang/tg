// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package contextkeys

// Key представляет типизированный ключ для контекста.
type Key string

const (
	// Force - ключ для передачи флага force через контекст.
	Force Key = "force"
	// Skipped - ключ для передачи флага skipped через контекст.
	Skipped Key = "skipped"
	// SkipDatabaseUpdate - ключ для пропуска обновления базы данных через контекст.
	SkipDatabaseUpdate Key = "skipDatabaseUpdate"
	// Source - ключ для передачи source через контекст.
	Source Key = "source"
)
