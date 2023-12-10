package types

type Constant struct {
	Base
	Iota      bool
	Value     interface{}
	Constants []Constant
	Type      Type `json:"type,omitempty"`
}

// String representation of variable without docs
func (v Constant) String() string {
	return v.Name + " " + v.Type.String()
}

func (v Constant) GoString() string {
	return v.String()
}
