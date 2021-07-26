package generator

import (
	"bytes"
	"fmt"
)

type bytesWriter struct {
	bytes.Buffer
}

func (w *bytesWriter) Line() {
	w.add("\n")
}

func (w *bytesWriter) add(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(&w.Buffer, format, args...)
}
