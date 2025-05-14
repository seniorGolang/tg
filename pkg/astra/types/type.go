package types

import (
	"strconv"
	"strings"
)

type Type interface {
	String() string
	t()
}

type LinearType interface {
	NextType() Type
}

type TInterface struct {
	Interface *Interface `json:"interface,omitempty"`
}

func (i TInterface) t() {}

func (i TInterface) String() string {
	if i.Interface != nil {
		return i.Interface.String()
	}
	return ""
}

type TMap struct {
	Key   Type `json:"key,omitempty"`
	Value Type `json:"value,omitempty"`
}

func (m TMap) t() {}

func (m TMap) String() string {
	return "map[" + m.Key.String() + "]" + m.Value.String()
}

type TName struct {
	TypeName string `json:"type_name,omitempty"`
}

func (i TName) t() {}

func (i TName) String() string {
	return i.TypeName
}

func (i TName) NextType() Type {
	return nil
}

type TPointer struct {
	NumberOfPointers int  `json:"number_of_pointers,omitempty"`
	Next             Type `json:"next,omitempty"`
}

func (i TPointer) t() {}

func (i TPointer) String() string {
	str := strings.Repeat("*", i.NumberOfPointers)
	if i.Next != nil {
		str += i.Next.String()
	}
	return str
}

func (i TPointer) NextType() Type {
	return i.Next
}

type TArray struct {
	ArrayLen   int  `json:"array_len,omitempty"`
	IsSlice    bool `json:"is_slice,omitempty"` // [] declaration
	IsEllipsis bool `json:"is_ellipsis,omitempty"`
	Next       Type `json:"next,omitempty"`
}

func (i TArray) t() {}

func (i TArray) String() string {
	str := ""
	switch {
	case i.IsEllipsis:
		str += "..."
	case i.IsSlice:
		str += "[]"
	default:
		str += "[" + strconv.Itoa(i.ArrayLen) + "]"
	}
	if i.Next != nil {
		str += i.Next.String()
	}
	return str
}

func (i TArray) NextType() Type {
	return i.Next
}

type TImport struct {
	Import *Import `json:"import,omitempty"`
	Next   Type    `json:"next,omitempty"`
}

func (i TImport) t() {}

func (i TImport) String() string {
	str := ""
	if i.Import != nil {
		str += i.Import.Name + "."
	}
	if i.Next != nil {
		str += i.Next.String()
	}
	return str
}

func (i TImport) NextType() Type {
	return i.Next
}

// TEllipsis used only for function params in declarations like `strs ...string`
type TEllipsis struct {
	Next Type `json:"next,omitempty"`
}

func (i TEllipsis) t() {}

func (i TEllipsis) String() string {
	str := "..."
	if i.Next != nil {
		return str + i.Next.String()
	}
	return str
}

func (i TEllipsis) NextType() Type {
	return i.Next
}

const (
	ChanDirSend = 1
	ChanDirRecv = 2
	ChanDirAny  = ChanDirSend | ChanDirRecv
)

type TChan struct {
	Direction int  `json:"direction"`
	Next      Type `json:"next"`
}

func (c TChan) t() {}

func (c TChan) NextType() Type {
	return c.Next
}

var strForChan = map[int]string{
	ChanDirSend: "chan<-",
	ChanDirRecv: "<-chan",
	ChanDirAny:  "chan",
}

func (c TChan) String() string {
	str := strForChan[c.Direction]
	if c.Next != nil {
		return str + " " + c.Next.String()
	}
	return str
}
