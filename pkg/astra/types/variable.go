package types

type Variable struct {
	Base
	Type Type `json:"type,omitempty"`
}

// String representation of variable without docs
func (v Variable) String() string {
	return v.Name + " " + v.Type.String()
}

func (v Variable) GoString() string {
	return v.String()
}
