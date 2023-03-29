package viewer

import (
	"fmt"
)

func Sprintln(a ...interface{}) string {
	return fmt.Sprintln(convertArgs(a)...)
}

func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, convertArgs(a)...)
}

func Sprint(a ...interface{}) string {
	return fmt.Sprint(convertArgs(a)...)
}

func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Printf(format, convertArgs(a)...)
}

func convertArgs(args []interface{}) (formatters []interface{}) {
	formatters = make([]interface{}, len(args))
	for index, arg := range args {
		formatters[index] = NewFormatter(arg)
	}
	return formatters
}

func newFormatter(cs *ConfigState, v interface{}) fmt.Formatter {
	fs := &formatState{value: v, cs: cs}
	fs.pointers = make(map[uintptr]int)
	return fs
}

func NewFormatter(v interface{}) fmt.Formatter {
	return newFormatter(&Config, v)
}
