// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (logger.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package logger

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/v2/pkg/logger/format"
)

var Log Logger

type Logger = *logrus.Entry

func init() {
	Log = logrus.WithTime(time.Now())
	logrus.SetFormatter(&format.Formatter{TimestampFormat: time.StampMilli})
}
