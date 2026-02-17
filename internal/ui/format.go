// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package ui

import (
	"fmt"
	"time"
)

func formatDurationFixed(d time.Duration) (result string) {

	if d < time.Second {
		ms := d.Milliseconds()
		str := fmt.Sprintf(timeFormatMs, ms)
		if len(str) > timeWidth {
			str = str[:timeWidth]
		}
		return fmt.Sprintf(widthFormat, timeWidth, str)
	}

	if d < time.Minute {
		sec := int(d.Seconds())
		str := fmt.Sprintf(timeFormatS, sec)
		return fmt.Sprintf(widthFormat, timeWidth, str)
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	var str string
	if seconds == 0 {
		str = fmt.Sprintf(timeFormatM, minutes)
	} else {
		str = fmt.Sprintf(timeFormatMS, minutes, seconds)
		if len(str) > timeWidth {
			str = str[:timeWidth]
		}
	}

	return fmt.Sprintf(widthFormat, timeWidth, str)
}
