package viewer

import (
	"strconv"
	"strings"
)

type option func([]byte) []byte

func applyOptions(bytes []byte, opts ...option) (view []byte) {
	view = make([]byte, len(bytes))
	copy(view, bytes)
	for _, opt := range opts {
		if opt != nil {
			view = opt(view)
		}
	}
	return
}

func hide(formula string) option {

	return func(bytes []byte) (view []byte) {

		var f, t int64
		if formula == "fh" {
			t = int64(len(bytes) / 2)
		} else if formula == "lh" {
			f = int64(len(bytes) / 2)
		} else if formula == "md" {
			f = int64(len(bytes) / 3)
			t = int64(len(bytes) - len(bytes)/3)
		} else if strings.Contains(formula, ":") {
			params := strings.Split(formula, ":")
			if len(params) == 2 {
				f, _ = strconv.ParseInt(params[0], 10, 32)
				t, _ = strconv.ParseInt(params[1], 10, 32)
			}
		}
		if formula != "-" {
			view = make([]byte, len(bytes))
			copy(view, bytes)
			view = append(view[:f], []byte(strings.Repeat("*", len(bytes)-int(f)))...)
			if t != 0 {
				view = append(view[:t], bytes[t:]...)
			}
		}
		return
	}
}
