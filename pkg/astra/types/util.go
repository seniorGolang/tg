package types

// List of all builtin types.
var BuiltinTypes = map[string]bool{
	"bool":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
	"string":     true,
	"int":        true,
	"uint":       true,
	"uintptr":    true,
	"byte":       true,
	"rune":       true,
	"error":      true,
	"any":        true,
}

// List of all builtin functions.
var BuiltinFunctions = map[string]bool{
	"append":  true,
	"copy":    true,
	"delete":  true,
	"len":     true,
	"cap":     true,
	"make":    true,
	"new":     true,
	"complex": true,
	"real":    true,
	"imag":    true,
	"close":   true,
	"panic":   true,
	"recover": true,
	"print":   true,
	"println": true,
}

// Checks is type is builtin type.
func IsBuiltin(t Type) bool {
	name, ok := t.(TName)
	return ok && IsBuiltinTypeString(name.TypeName)
}

func IsBuiltinTypeString(t string) bool {
	return BuiltinTypes[t]
}

func IsBuiltinFuncString(t string) bool {
	return BuiltinFunctions[t]
}

func IsBuiltinString(t string) bool {
	return IsBuiltinTypeString(t) || IsBuiltinFuncString(t)
}

// Returns name of type if it has it.
// Raw maps and interfaces do not have names.
func TypeName(t Type) *string {
	for {
		switch tt := t.(type) {
		case TName:
			return &tt.TypeName
		case TInterface:
			return nil
		case TMap:
			return nil
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

// Returns Import of type or nil.
func TypeImport(t Type) *Import {
	for {
		switch tt := t.(type) {
		case TImport:
			return tt.Import
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

// Returns first array entity of type.
// If array not found, returns nil.
func TypeArray(t Type) Type {
	for {
		switch tt := t.(type) {
		case TArray:
			return tt
		case TInterface:
			return nil
		case TMap:
			return nil
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

func TypeMap(t Type) Type {
	for {
		switch tt := t.(type) {
		case TInterface:
			return nil
		case TMap:
			return tt
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

func TypeInterface(t Type) Type {
	for {
		switch tt := t.(type) {
		case TInterface:
			return tt
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

func TypeEllipsis(t Type) Type {
	for {
		switch tt := t.(type) {
		case TEllipsis:
			return tt
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

func TypeStruct(t Type) Type {
	for {
		switch tt := t.(type) {
		case Struct:
			return tt
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

func TypeFunction(t Type) Type {
	for {
		switch tt := t.(type) {
		case Function:
			return tt
		default:
			next, ok := tt.(LinearType)
			if !ok {
				return nil
			}
			t = next.NextType()
		}
	}
}

func IsType(f func(Type) Type) func(Type) bool {
	return func(t Type) bool {
		return f(t) != nil
	}
}

// Checks, is type contain some type.
// Generic checkers.
var (
	// Checks, is type contain array.
	IsArray = IsType(TypeArray)
	// Checks, is type contain map.
	IsMap = IsType(TypeMap)
	// Checks, is type contain interface.
	IsInterface = IsType(TypeInterface)
	IsEllipsis  = IsType(TypeEllipsis)
	IsStruct    = IsType(TypeStruct)
	IsFunction  = IsType(TypeFunction)
)
