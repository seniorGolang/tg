package types

import (
	"fmt"
	"strings"
)

type Interface struct {
	Base
	Methods    []*Function `json:"methods,omitempty"`    // List of functions (methods) of the interface.
	Interfaces []Variable  `json:"interfaces,omitempty"` // List of embedded interfaces.
}

func (i Interface) String() string {
	methods := make([]string, len(i.Methods)+len(i.Interfaces))
	for k, m := range i.Methods {
		methods[k] = m.funcStr()
	}
	n := len(i.Methods)
	for k, m := range i.Interfaces {
		methods[n+k] = m.String()
	}
	return fmt.Sprintf("type %s interface {\n\t%s\n}", i.Name, strings.Join(methods, "\n\t"))
}

func (i Interface) GoString() string {
	return i.String()
}

func (i Interface) IsEmpty() bool {
	return len(i.Methods) == 0 && len(i.Interfaces) == 0
}
