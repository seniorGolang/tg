package typescript

import (
	"bytes"
)

// NewFile Creates a new file, with the specified package name.
func NewFile() *File {
	return &File{
		Group: &Group{
			multi: true,
		},
		imports: map[string]importdef{},
		hints:   map[string]importdef{},
	}
}

// NewFilePath creates a new file while specifying the package path - the
// package name is inferred from the path.
func NewFilePath(packagePath string) *File {
	return &File{
		Group: &Group{
			multi: true,
		},
		path:    packagePath,
		imports: map[string]importdef{},
		hints:   map[string]importdef{},
	}
}

// NewFilePathName creates a new file with the specified package path and name.
func NewFilePathName(packagePath, packageName string) *File {
	return &File{
		Group: &Group{
			multi: true,
		},
		path:    packagePath,
		imports: map[string]importdef{},
		hints:   map[string]importdef{},
	}
}

// File represents a single source file. Package imports are managed
// automaticaly by File.
type File struct {
	*Group
	path        string
	imports     map[string]importdef
	hints       map[string]importdef
	comments    []string
	headers     []string
	cgoPreamble []string
	// If you're worried about generated package aliases conflicting with local variable names, you
	// can set a prefix here. Package foo becomes {prefix}_foo.
	PackagePrefix string
	// CanonicalPath adds a canonical import path annotation to the package clause.
	CanonicalPath string
}

type importdef struct {
	from  string
	items []string
}

// HeaderComment adds a comment to the top of the file, above any package
// comments. A blank line is rendered below the header comments, ensuring
// header comments are not included in the package doc.
func (f *File) HeaderComment(comment string) {
	f.headers = append(f.headers, comment)
}

// PackageComment adds a comment to the top of the file, above the package
// keyword.
func (f *File) PackageComment(comment string) {
	f.comments = append(f.comments, comment)
}

// CgoPreamble adds a cgo preamble comment that is rendered directly before the "C" pseudo-package
// import.
func (f *File) CgoPreamble(comment string) {
	f.cgoPreamble = append(f.cgoPreamble, comment)
}

func (f *File) Import(from string, items ...string) {
	f.imports[from] = importdef{from: from, items: items}
}

func (f *File) isLocal(path string) bool {
	return f.path == path
}

var reserved = []string{
	/* keywords */
	"break", "default", "func", "interface", "select", "case", "defer", "go", "map", "struct", "chan", "else", "goto", "package", "switch", "const", "fallthrough", "if", "range", "type", "continue", "for", "import", "return", "var",
	/* predeclared */
	"bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "true", "false", "iota", "nil", "append", "cap", "close", "complex", "copy", "delete", "imag", "len", "make", "new", "panic", "print", "println", "real", "recover",
	/* common variables */
	"err",
}

func isReservedWord(alias string) bool {
	for _, name := range reserved {
		if alias == name {
			return true
		}
	}
	return false
}

func (f *File) isValidAlias(alias string) bool {
	// multiple dot-imports are ok
	if alias == "." {
		return true
	}
	// the import alias is invalid if it's a reserved word
	if isReservedWord(alias) {
		return false
	}
	return true
}

// GoString renders the File for testing. Any error will cause a panic.
func (f *File) GoString() string {
	buf := &bytes.Buffer{}
	if err := f.Render(buf); err != nil {
		panic(err)
	}
	return buf.String()
}
