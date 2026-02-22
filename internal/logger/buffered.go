// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package logger

import (
	"bytes"

	"github.com/seniorGolang/tg/v3/internal/plugin"

	"github.com/pterm/pterm"
)

type BufferedLoggerAdapter struct {
	buffer *LogBuffer
	logger *pterm.Logger
}

func NewBufferedLoggerAdapter(buffer *LogBuffer) (adapter *BufferedLoggerAdapter) {

	logger := pterm.DefaultLogger.
		WithCaller(false).
		WithTime(false)

	return &BufferedLoggerAdapter{
		buffer: buffer,
		logger: logger,
	}
}

func (a *BufferedLoggerAdapter) addFormattedLog(formattedLine string) {

	a.buffer.mu.Lock()
	defer a.buffer.mu.Unlock()

	if len(formattedLine) > 0 && formattedLine[len(formattedLine)-1] == newlineChar {
		formattedLine = formattedLine[:len(formattedLine)-1]
	}
	a.buffer.logs = append(a.buffer.logs, formattedLine)
}

func (a *BufferedLoggerAdapter) log(level string, msg string, args ...any) {

	var loggerArgs []pterm.LoggerArgument
	if len(args) > 0 {
		loggerArgs = make([]pterm.LoggerArgument, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				var ok bool
				var key string
				key, ok = args[i].(string)
				if !ok {
					continue
				}
				loggerArgs = append(loggerArgs, pterm.LoggerArgument{
					Key:   key,
					Value: args[i+1],
				})
			}
		}
	}

	var buf bytes.Buffer
	tempLogger := a.logger.WithWriter(&buf)

	logWithLevelString(tempLogger, level, msg, loggerArgs)

	a.addFormattedLog(buf.String())
}

func logWithLevelString(logger *pterm.Logger, level string, msg string, args []pterm.LoggerArgument) {

	if len(args) > 0 {
		switch level {
		case logLevelDebug:
			logger.Debug(msg, args)
		case logLevelInfo:
			logger.Info(msg, args)
		case logLevelWarn:
			logger.Warn(msg, args)
		case logLevelError:
			logger.Error(msg, args)
		}
	} else {
		switch level {
		case logLevelDebug:
			logger.Debug(msg)
		case logLevelInfo:
			logger.Info(msg)
		case logLevelWarn:
			logger.Warn(msg)
		case logLevelError:
			logger.Error(msg)
		}
	}
}

func (a *BufferedLoggerAdapter) Debug(msg string, args ...any) {

	a.log(logLevelDebug, msg, args...)
}

func (a *BufferedLoggerAdapter) Info(msg string, args ...any) {

	a.log(logLevelInfo, msg, args...)
}

func (a *BufferedLoggerAdapter) Warn(msg string, args ...any) {

	a.log(logLevelWarn, msg, args...)
}

func (a *BufferedLoggerAdapter) Error(msg string, args ...any) {

	a.log(logLevelError, msg, args...)
}

var _ plugin.Logger = (*BufferedLoggerAdapter)(nil)
