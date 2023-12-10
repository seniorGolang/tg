package types

type FileType struct {
	Base
	Type    Type      `json:"type,omitempty"`
	Methods []*Method `json:"methods,omitempty"`
}

// File is a top-level entity, that contains all top-level declarations of the file.
type File struct {
	Base                   // `File.Name` is package name, `File.Docs` is a comments above `package ...`
	Imports    []*Import   `json:"imports,omitempty"`    // Contains imports and their aliases from `import` blocks.
	Constants  []Constant  `json:"constants,omitempty"`  // Contains constant variables from `const` blocks.
	Vars       []Variable  `json:"vars,omitempty"`       // Contains variables from `var` blocks.
	Interfaces []Interface `json:"interfaces,omitempty"` // Contains `type Foo interface` declarations.
	Structures []Struct    `json:"structures,omitempty"` // Contains `type Foo struct` declarations.
	Functions  []Function  `json:"functions,omitempty"`  // Contains `func Foo() {}` declarations.
	Methods    []Method    `json:"methods,omitempty"`    // Contains `func (a A) Foo(b B) (c C) {}` declarations.
	Types      []FileType  `json:"types,omitempty"`      // Contains `type X int` declarations.
}

func (f File) HasPackage(packageName string) bool {
	for i := range f.Imports {
		if f.Imports[i] != nil && f.Imports[i].Package == packageName {
			return true
		}
	}
	return false
}
