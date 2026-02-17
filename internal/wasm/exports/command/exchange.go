// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package command

// CommandResponse представляет результат выполнения команды через хост.
// Stdout и Stderr передаются через потоки (streamID), а не кодируются.
type CommandResponse struct {
	ExitCode       int    `json:"exitCode"`
	StdoutStreamID int32  `json:"stdoutStreamID,omitempty"`
	StderrStreamID int32  `json:"stderrStreamID,omitempty"`
	Error          string `json:"error,omitempty"`
}
