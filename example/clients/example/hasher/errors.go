package hasher

import (
	"fmt"
)

type ErrNotStringer struct {
	Field string
}

func (ens *ErrNotStringer) Error() string {
	return fmt.Sprintf("hashstructure: %s has hash:\"string\" set, but does not implement fmt.Stringer", ens.Field)
}
