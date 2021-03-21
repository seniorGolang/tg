package types

import "fmt"

type Import struct {
	Base
	Package string `json:"package,omitempty"`
}

func (i Import) String() string {
	return fmt.Sprintf("%s \"%s\"", i.Name, i.Package)
}

func (i Import) GoString() string {
	return i.String()
}
