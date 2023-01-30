package jsonrpc

import (
	"fmt"
)

func toString(v interface{}) string {

	switch s := v.(type) {
	case string:
		return s
	case fmt.Stringer:
		return s.String()
	}
	return ""
}
