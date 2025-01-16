package types

import (
	"fmt"
	"strings"
)

type Function struct {
	Base
	Args    []Variable `json:"args,omitempty"`
	Results []Variable `json:"results,omitempty"`
}

type Method struct {
	Function
	Receiver Variable `json:"receiver,omitempty"`
}

func (f Function) funcStr() string {

	var args, results = make([]string, 0, len(f.Args)), make([]string, 0, len(f.Args))
	for _, arg := range f.Args {
		args = append(args, arg.String())
	}
	for _, res := range f.Results {
		results = append(results, res.String())
	}
	return fmt.Sprintf("%s(%s) (%s)", f.Name, strings.Join(args, ", "), strings.Join(results, ", "))
}

func (f Function) String() string {
	return "func " + f.funcStr()
}

func (f Function) GoString() string {
	return f.String()
}

func (f Function) t() {}

func (f Method) String() string {
	return fmt.Sprintf("func (%s) %s", f.Receiver.String(), f.funcStr())
}

func (f Method) GoString() string {
	return f.String()
}
