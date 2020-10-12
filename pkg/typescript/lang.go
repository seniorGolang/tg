package typescript

// Parents renders a single item in parenthesis. Use for type conversion or to specify evaluation order.
func Parents(item Code) *Statement {
	return newStatement().Parents(item)
}

// Parents renders a single item in parenthesis. Use for type conversion or to specify evaluation order.
func (g *Group) Parents(item Code) *Statement {
	s := Parents(item)
	g.items = append(g.items, s)
	return s
}

// Parents renders a single item in parenthesis. Use for type conversion or to specify evaluation order.
func (s *Statement) Parents(item Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{item},
		multi:     false,
		name:      "parents",
		open:      "(",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// List renders a comma separated list. Use for multiple return functions.
func List(items ...Code) *Statement {
	return newStatement().List(items...)
}

// List renders a comma separated list. Use for multiple return functions.
func (g *Group) List(items ...Code) *Statement {
	s := List(items...)
	g.items = append(g.items, s)
	return s
}

// List renders a comma separated list. Use for multiple return functions.
func (s *Statement) List(items ...Code) *Statement {
	g := &Group{
		close:     "",
		items:     items,
		multi:     false,
		name:      "list",
		open:      "",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// ListFunc renders a comma separated list. Use for multiple return functions.
func ListFunc(f func(*Group)) *Statement {
	return newStatement().ListFunc(f)
}

// ListFunc renders a comma separated list. Use for multiple return functions.
func (g *Group) ListFunc(f func(*Group)) *Statement {
	s := ListFunc(f)
	g.items = append(g.items, s)
	return s
}

// ListFunc renders a comma separated list. Use for multiple return functions.
func (s *Statement) ListFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "",
		multi:     false,
		name:      "list",
		open:      "",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Values renders a comma separated list enclosed by curly braces. Use for slice or composite literals.
func Values(values ...Code) *Statement {
	return newStatement().Values(values...)
}

// Values renders a comma separated list enclosed by curly braces. Use for slice or composite literals.
func (g *Group) Values(values ...Code) *Statement {
	s := Values(values...)
	g.items = append(g.items, s)
	return s
}

// Values renders a comma separated list enclosed by curly braces. Use for slice or composite literals.
func (s *Statement) Values(values ...Code) *Statement {
	g := &Group{
		close:     "}",
		items:     values,
		multi:     false,
		name:      "values",
		open:      "{",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// ValuesFunc renders a comma separated list enclosed by curly braces. Use for slice or composite literals.
func ValuesFunc(f func(*Group)) *Statement {
	return newStatement().ValuesFunc(f)
}

// ValuesFunc renders a comma separated list enclosed by curly braces. Use for slice or composite literals.
func (g *Group) ValuesFunc(f func(*Group)) *Statement {
	s := ValuesFunc(f)
	g.items = append(g.items, s)
	return s
}

// ValuesFunc renders a comma separated list enclosed by curly braces. Use for slice or composite literals.
func (s *Statement) ValuesFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "}",
		multi:     false,
		name:      "values",
		open:      "{",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Index renders a colon separated list enclosed by square brackets. Use for array / slice indexes and definitions.
func Index(items ...Code) *Statement {
	return newStatement().Index(items...)
}

// Index renders a colon separated list enclosed by square brackets. Use for array / slice indexes and definitions.
func (g *Group) Index(items ...Code) *Statement {
	s := Index(items...)
	g.items = append(g.items, s)
	return s
}

// Index renders a colon separated list enclosed by square brackets. Use for array / slice indexes and definitions.
func (s *Statement) Index(items ...Code) *Statement {
	g := &Group{
		close:     "]",
		items:     items,
		multi:     false,
		name:      "index",
		open:      "[",
		separator: ":",
	}
	*s = append(*s, g)
	return s
}

// IndexFunc renders a colon separated list enclosed by square brackets. Use for array / slice indexes and definitions.
func IndexFunc(f func(*Group)) *Statement {
	return newStatement().IndexFunc(f)
}

// IndexFunc renders a colon separated list enclosed by square brackets. Use for array / slice indexes and definitions.
func (g *Group) IndexFunc(f func(*Group)) *Statement {
	s := IndexFunc(f)
	g.items = append(g.items, s)
	return s
}

// IndexFunc renders a colon separated list enclosed by square brackets. Use for array / slice indexes and definitions.
func (s *Statement) IndexFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "]",
		multi:     false,
		name:      "index",
		open:      "[",
		separator: ":",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Block renders a statement list enclosed by curly braces. Use for code blocks. A special case applies when used directly after Case or Default, where the braces are omitted. This allows use in switch and select statements.
func Block(statements ...Code) *Statement {
	return newStatement().Block(statements...)
}

// Block renders a statement list enclosed by curly braces. Use for code blocks. A special case applies when used directly after Case or Default, where the braces are omitted. This allows use in switch and select statements.
func (g *Group) Block(statements ...Code) *Statement {
	s := Block(statements...)
	g.items = append(g.items, s)
	return s
}

// Block renders a statement list enclosed by curly braces. Use for code blocks. A special case applies when used directly after Case or Default, where the braces are omitted. This allows use in switch and select statements.
func (s *Statement) Block(statements ...Code) *Statement {
	g := &Group{
		close:     "}",
		items:     statements,
		multi:     true,
		name:      "block",
		open:      "{",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// BlockFunc renders a statement list enclosed by curly braces. Use for code blocks. A special case applies when used directly after Case or Default, where the braces are omitted. This allows use in switch and select statements.
func BlockFunc(f func(*Group)) *Statement {
	return newStatement().BlockFunc(f)
}

// BlockFunc renders a statement list enclosed by curly braces. Use for code blocks. A special case applies when used directly after Case or Default, where the braces are omitted. This allows use in switch and select statements.
func (g *Group) BlockFunc(f func(*Group)) *Statement {
	s := BlockFunc(f)
	g.items = append(g.items, s)
	return s
}

// BlockFunc renders a statement list enclosed by curly braces. Use for code blocks. A special case applies when used directly after Case or Default, where the braces are omitted. This allows use in switch and select statements.
func (s *Statement) BlockFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "}",
		multi:     true,
		name:      "block",
		open:      "{",
		separator: "",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Defs renders a statement list enclosed in parenthesis. Use for definition lists.
func Defs(definitions ...Code) *Statement {
	return newStatement().Defs(definitions...)
}

// Defs renders a statement list enclosed in parenthesis. Use for definition lists.
func (g *Group) Defs(definitions ...Code) *Statement {
	s := Defs(definitions...)
	g.items = append(g.items, s)
	return s
}

// Defs renders a statement list enclosed in parenthesis. Use for definition lists.
func (s *Statement) Defs(definitions ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     definitions,
		multi:     true,
		name:      "defs",
		open:      "(",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// DefsFunc renders a statement list enclosed in parenthesis. Use for definition lists.
func DefsFunc(f func(*Group)) *Statement {
	return newStatement().DefsFunc(f)
}

// DefsFunc renders a statement list enclosed in parenthesis. Use for definition lists.
func (g *Group) DefsFunc(f func(*Group)) *Statement {
	s := DefsFunc(f)
	g.items = append(g.items, s)
	return s
}

// DefsFunc renders a statement list enclosed in parenthesis. Use for definition lists.
func (s *Statement) DefsFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ")",
		multi:     true,
		name:      "defs",
		open:      "(",
		separator: "",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Call renders a comma separated list enclosed by parenthesis. Use for function calls.
func Call(params ...Code) *Statement {
	return newStatement().Call(params...)
}

// Call renders a comma separated list enclosed by parenthesis. Use for function calls.
func (g *Group) Call(params ...Code) *Statement {
	s := Call(params...)
	g.items = append(g.items, s)
	return s
}

// Call renders a comma separated list enclosed by parenthesis. Use for function calls.
func (s *Statement) Call(params ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     params,
		multi:     false,
		name:      "call",
		open:      "(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// CallFunc renders a comma separated list enclosed by parenthesis. Use for function calls.
func CallFunc(f func(*Group)) *Statement {
	return newStatement().CallFunc(f)
}

// CallFunc renders a comma separated list enclosed by parenthesis. Use for function calls.
func (g *Group) CallFunc(f func(*Group)) *Statement {
	s := CallFunc(f)
	g.items = append(g.items, s)
	return s
}

// CallFunc renders a comma separated list enclosed by parenthesis. Use for function calls.
func (s *Statement) CallFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ")",
		multi:     false,
		name:      "call",
		open:      "(",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Params renders a comma separated list enclosed by parenthesis. Use for function parameters and method receivers.
func Params(params ...Code) *Statement {
	return newStatement().Params(params...)
}

// Params renders a comma separated list enclosed by parenthesis. Use for function parameters and method receivers.
func (g *Group) Params(params ...Code) *Statement {
	s := Params(params...)
	g.items = append(g.items, s)
	return s
}

// Params renders a comma separated list enclosed by parenthesis. Use for function parameters and method receivers.
func (s *Statement) Params(params ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     params,
		multi:     false,
		name:      "params",
		open:      "(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// ParamsFunc renders a comma separated list enclosed by parenthesis. Use for function parameters and method receivers.
func ParamsFunc(f func(*Group)) *Statement {
	return newStatement().ParamsFunc(f)
}

// ParamsFunc renders a comma separated list enclosed by parenthesis. Use for function parameters and method receivers.
func (g *Group) ParamsFunc(f func(*Group)) *Statement {
	s := ParamsFunc(f)
	g.items = append(g.items, s)
	return s
}

// ParamsFunc renders a comma separated list enclosed by parenthesis. Use for function parameters and method receivers.
func (s *Statement) ParamsFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ")",
		multi:     false,
		name:      "params",
		open:      "(",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Assert renders a period followed by a single item enclosed by parenthesis. Use for type assertions.
func Assert(typ Code) *Statement {
	return newStatement().Assert(typ)
}

// Assert renders a period followed by a single item enclosed by parenthesis. Use for type assertions.
func (g *Group) Assert(typ Code) *Statement {
	s := Assert(typ)
	g.items = append(g.items, s)
	return s
}

// Assert renders a period followed by a single item enclosed by parenthesis. Use for type assertions.
func (s *Statement) Assert(typ Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{typ},
		multi:     false,
		name:      "assert",
		open:      ".(",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// Map renders the keyword followed by a single item enclosed by square brackets. Use for map definitions.
func Map(typ Code) *Statement {
	return newStatement().Map(typ)
}

// Map renders the keyword followed by a single item enclosed by square brackets. Use for map definitions.
func (g *Group) Map(typ Code) *Statement {
	s := Map(typ)
	g.items = append(g.items, s)
	return s
}

// Map renders the keyword followed by a single item enclosed by square brackets. Use for map definitions.
func (s *Statement) Map(typ Code) *Statement {
	g := &Group{
		close:     "]",
		items:     []Code{typ},
		multi:     false,
		name:      "map",
		open:      "map[",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// If renders the keyword followed by a semicolon separated list.
func If(conditions ...Code) *Statement {
	return newStatement().If(conditions...)
}

// If renders the keyword followed by a semicolon separated list.
func (g *Group) If(conditions ...Code) *Statement {
	s := If(conditions...)
	g.items = append(g.items, s)
	return s
}

// If renders the keyword followed by a semicolon separated list.
func (s *Statement) If(conditions ...Code) *Statement {
	g := &Group{
		close:     "",
		items:     conditions,
		multi:     false,
		name:      "if",
		open:      "if ",
		separator: ";",
	}
	*s = append(*s, g)
	return s
}

// IfFunc renders the keyword followed by a semicolon separated list.
func IfFunc(f func(*Group)) *Statement {
	return newStatement().IfFunc(f)
}

// IfFunc renders the keyword followed by a semicolon separated list.
func (g *Group) IfFunc(f func(*Group)) *Statement {
	s := IfFunc(f)
	g.items = append(g.items, s)
	return s
}

// IfFunc renders the keyword followed by a semicolon separated list.
func (s *Statement) IfFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "",
		multi:     false,
		name:      "if",
		open:      "if ",
		separator: ";",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Return renders the keyword followed by a comma separated list.
func Return(results ...Code) *Statement {
	return newStatement().Return(results...)
}

// Return renders the keyword followed by a comma separated list.
func (g *Group) Return(results ...Code) *Statement {
	s := Return(results...)
	g.items = append(g.items, s)
	return s
}

// Return renders the keyword followed by a comma separated list.
func (s *Statement) Return(results ...Code) *Statement {
	g := &Group{
		close:     "",
		items:     results,
		multi:     false,
		name:      "return",
		open:      "return ",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// ReturnFunc renders the keyword followed by a comma separated list.
func ReturnFunc(f func(*Group)) *Statement {
	return newStatement().ReturnFunc(f)
}

// ReturnFunc renders the keyword followed by a comma separated list.
func (g *Group) ReturnFunc(f func(*Group)) *Statement {
	s := ReturnFunc(f)
	g.items = append(g.items, s)
	return s
}

// ReturnFunc renders the keyword followed by a comma separated list.
func (s *Statement) ReturnFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "",
		multi:     false,
		name:      "return",
		open:      "return ",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// For renders the keyword followed by a semicolon separated list.
func For(conditions ...Code) *Statement {
	return newStatement().For(conditions...)
}

// For renders the keyword followed by a semicolon separated list.
func (g *Group) For(conditions ...Code) *Statement {
	s := For(conditions...)
	g.items = append(g.items, s)
	return s
}

// For renders the keyword followed by a semicolon separated list.
func (s *Statement) For(conditions ...Code) *Statement {
	g := &Group{
		close:     "",
		items:     conditions,
		multi:     false,
		name:      "for",
		open:      "for ",
		separator: ";",
	}
	*s = append(*s, g)
	return s
}

// ForFunc renders the keyword followed by a semicolon separated list.
func ForFunc(f func(*Group)) *Statement {
	return newStatement().ForFunc(f)
}

// ForFunc renders the keyword followed by a semicolon separated list.
func (g *Group) ForFunc(f func(*Group)) *Statement {
	s := ForFunc(f)
	g.items = append(g.items, s)
	return s
}

// ForFunc renders the keyword followed by a semicolon separated list.
func (s *Statement) ForFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "",
		multi:     false,
		name:      "for",
		open:      "for ",
		separator: ";",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Switch renders the keyword followed by a semicolon separated list.
func Switch(conditions ...Code) *Statement {
	return newStatement().Switch(conditions...)
}

// Switch renders the keyword followed by a semicolon separated list.
func (g *Group) Switch(conditions ...Code) *Statement {
	s := Switch(conditions...)
	g.items = append(g.items, s)
	return s
}

// Switch renders the keyword followed by a semicolon separated list.
func (s *Statement) Switch(conditions ...Code) *Statement {
	g := &Group{
		close:     "",
		items:     conditions,
		multi:     false,
		name:      "switch",
		open:      "switch ",
		separator: ";",
	}
	*s = append(*s, g)
	return s
}

// SwitchFunc renders the keyword followed by a semicolon separated list.
func SwitchFunc(f func(*Group)) *Statement {
	return newStatement().SwitchFunc(f)
}

// SwitchFunc renders the keyword followed by a semicolon separated list.
func (g *Group) SwitchFunc(f func(*Group)) *Statement {
	s := SwitchFunc(f)
	g.items = append(g.items, s)
	return s
}

// SwitchFunc renders the keyword followed by a semicolon separated list.
func (s *Statement) SwitchFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "",
		multi:     false,
		name:      "switch",
		open:      "switch ",
		separator: ";",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Interface renders the keyword followed by a method list enclosed by curly braces.
func Interface(methods ...Code) *Statement {
	return newStatement().Interface(methods...)
}

// Interface renders the keyword followed by a method list enclosed by curly braces.
func (g *Group) Interface(methods ...Code) *Statement {
	s := Interface(methods...)
	g.items = append(g.items, s)
	return s
}

// Interface renders the keyword followed by a method list enclosed by curly braces.
func (s *Statement) Interface(methods ...Code) *Statement {
	g := &Group{
		close:     "}",
		items:     methods,
		multi:     true,
		name:      "interface",
		open:      "interface{",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// InterfaceFunc renders the keyword followed by a method list enclosed by curly braces.
func InterfaceFunc(f func(*Group)) *Statement {
	return newStatement().InterfaceFunc(f)
}

// InterfaceFunc renders the keyword followed by a method list enclosed by curly braces.
func (g *Group) InterfaceFunc(f func(*Group)) *Statement {
	s := InterfaceFunc(f)
	g.items = append(g.items, s)
	return s
}

// InterfaceFunc renders the keyword followed by a method list enclosed by curly braces.
func (s *Statement) InterfaceFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "}",
		multi:     true,
		name:      "interface",
		open:      "interface{",
		separator: "",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Struct renders the keyword followed by a field list enclosed by curly braces.
func Struct(fields ...Code) *Statement {
	return newStatement().Struct(fields...)
}

// Struct renders the keyword followed by a field list enclosed by curly braces.
func (g *Group) Struct(fields ...Code) *Statement {
	s := Struct(fields...)
	g.items = append(g.items, s)
	return s
}

// Struct renders the keyword followed by a field list enclosed by curly braces.
func (s *Statement) Struct(fields ...Code) *Statement {
	g := &Group{
		close:     "}",
		items:     fields,
		multi:     true,
		name:      "struct",
		open:      "struct{",
		separator: "",
	}
	*s = append(*s, g)
	return s
}

// StructFunc renders the keyword followed by a field list enclosed by curly braces.
func StructFunc(f func(*Group)) *Statement {
	return newStatement().StructFunc(f)
}

// StructFunc renders the keyword followed by a field list enclosed by curly braces.
func (g *Group) StructFunc(f func(*Group)) *Statement {
	s := StructFunc(f)
	g.items = append(g.items, s)
	return s
}

// StructFunc renders the keyword followed by a field list enclosed by curly braces.
func (s *Statement) StructFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     "}",
		multi:     true,
		name:      "struct",
		open:      "struct{",
		separator: "",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Case renders the keyword followed by a comma separated list.
func Case(cases ...Code) *Statement {
	return newStatement().Case(cases...)
}

// Case renders the keyword followed by a comma separated list.
func (g *Group) Case(cases ...Code) *Statement {
	s := Case(cases...)
	g.items = append(g.items, s)
	return s
}

// Case renders the keyword followed by a comma separated list.
func (s *Statement) Case(cases ...Code) *Statement {
	g := &Group{
		close:     ":",
		items:     cases,
		multi:     false,
		name:      "case",
		open:      "case ",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// CaseFunc renders the keyword followed by a comma separated list.
func CaseFunc(f func(*Group)) *Statement {
	return newStatement().CaseFunc(f)
}

// CaseFunc renders the keyword followed by a comma separated list.
func (g *Group) CaseFunc(f func(*Group)) *Statement {
	s := CaseFunc(f)
	g.items = append(g.items, s)
	return s
}

// CaseFunc renders the keyword followed by a comma separated list.
func (s *Statement) CaseFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ":",
		multi:     false,
		name:      "case",
		open:      "case ",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Append renders the append built-in function.
func Append(args ...Code) *Statement {
	return newStatement().Append(args...)
}

// Append renders the append built-in function.
func (g *Group) Append(args ...Code) *Statement {
	s := Append(args...)
	g.items = append(g.items, s)
	return s
}

// Append renders the append built-in function.
func (s *Statement) Append(args ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     args,
		multi:     false,
		name:      "append",
		open:      "append(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// AppendFunc renders the append built-in function.
func AppendFunc(f func(*Group)) *Statement {
	return newStatement().AppendFunc(f)
}

// AppendFunc renders the append built-in function.
func (g *Group) AppendFunc(f func(*Group)) *Statement {
	s := AppendFunc(f)
	g.items = append(g.items, s)
	return s
}

// AppendFunc renders the append built-in function.
func (s *Statement) AppendFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ")",
		multi:     false,
		name:      "append",
		open:      "append(",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Cap renders the cap built-in function.
func Cap(v Code) *Statement {
	return newStatement().Cap(v)
}

// Cap renders the cap built-in function.
func (g *Group) Cap(v Code) *Statement {
	s := Cap(v)
	g.items = append(g.items, s)
	return s
}

// Cap renders the cap built-in function.
func (s *Statement) Cap(v Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{v},
		multi:     false,
		name:      "cap",
		open:      "cap(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Close renders the close built-in function.
func Close(c Code) *Statement {
	return newStatement().Close(c)
}

// Close renders the close built-in function.
func (g *Group) Close(c Code) *Statement {
	s := Close(c)
	g.items = append(g.items, s)
	return s
}

// Close renders the close built-in function.
func (s *Statement) Close(c Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{c},
		multi:     false,
		name:      "close",
		open:      "close(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Complex renders the complex built-in function.
func Complex(r Code, i Code) *Statement {
	return newStatement().Complex(r, i)
}

// Complex renders the complex built-in function.
func (g *Group) Complex(r Code, i Code) *Statement {
	s := Complex(r, i)
	g.items = append(g.items, s)
	return s
}

// Complex renders the complex built-in function.
func (s *Statement) Complex(r Code, i Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{r, i},
		multi:     false,
		name:      "complex",
		open:      "complex(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Copy renders the copy built-in function.
func Copy(dst Code, src Code) *Statement {
	return newStatement().Copy(dst, src)
}

// Copy renders the copy built-in function.
func (g *Group) Copy(dst Code, src Code) *Statement {
	s := Copy(dst, src)
	g.items = append(g.items, s)
	return s
}

// Copy renders the copy built-in function.
func (s *Statement) Copy(dst Code, src Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{dst, src},
		multi:     false,
		name:      "copy",
		open:      "copy(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Delete renders the delete built-in function.
func Delete(m Code, key Code) *Statement {
	return newStatement().Delete(m, key)
}

// Delete renders the delete built-in function.
func (g *Group) Delete(m Code, key Code) *Statement {
	s := Delete(m, key)
	g.items = append(g.items, s)
	return s
}

// Delete renders the delete built-in function.
func (s *Statement) Delete(m Code, key Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{m, key},
		multi:     false,
		name:      "delete",
		open:      "delete(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Imag renders the imag built-in function.
func Imag(c Code) *Statement {
	return newStatement().Imag(c)
}

// Imag renders the imag built-in function.
func (g *Group) Imag(c Code) *Statement {
	s := Imag(c)
	g.items = append(g.items, s)
	return s
}

// Imag renders the imag built-in function.
func (s *Statement) Imag(c Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{c},
		multi:     false,
		name:      "imag",
		open:      "imag(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Len renders the len built-in function.
func Len(v Code) *Statement {
	return newStatement().Len(v)
}

// Len renders the len built-in function.
func (g *Group) Len(v Code) *Statement {
	s := Len(v)
	g.items = append(g.items, s)
	return s
}

// Len renders the len built-in function.
func (s *Statement) Len(v Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{v},
		multi:     false,
		name:      "len",
		open:      "len(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Make renders the make built-in function. The final parameter of the make function is optional, so it is represented by a variadic parameter list.
func Make(args ...Code) *Statement {
	return newStatement().Make(args...)
}

// Make renders the make built-in function. The final parameter of the make function is optional, so it is represented by a variadic parameter list.
func (g *Group) Make(args ...Code) *Statement {
	s := Make(args...)
	g.items = append(g.items, s)
	return s
}

// Make renders the make built-in function. The final parameter of the make function is optional, so it is represented by a variadic parameter list.
func (s *Statement) Make(args ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     args,
		multi:     false,
		name:      "make",
		open:      "make(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// New renders the new built-in function.
func New(typ Code) *Statement {
	return newStatement().New(typ)
}

// New renders the new built-in function.
func (g *Group) New(typ Code) *Statement {
	s := New(typ)
	g.items = append(g.items, s)
	return s
}

// New renders the new built-in function.
func (s *Statement) New(typ Code) *Statement {
	g := &Group{
		close:     "",
		items:     []Code{typ},
		multi:     false,
		name:      "new",
		open:      "new ",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Panic renders the panic built-in function.
func Panic(v Code) *Statement {
	return newStatement().Panic(v)
}

// Panic renders the panic built-in function.
func (g *Group) Panic(v Code) *Statement {
	s := Panic(v)
	g.items = append(g.items, s)
	return s
}

// Panic renders the panic built-in function.
func (s *Statement) Panic(v Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{v},
		multi:     false,
		name:      "panic",
		open:      "panic(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Print renders the print built-in function.
func Print(args ...Code) *Statement {
	return newStatement().Print(args...)
}

// Print renders the print built-in function.
func (g *Group) Print(args ...Code) *Statement {
	s := Print(args...)
	g.items = append(g.items, s)
	return s
}

// Print renders the print built-in function.
func (s *Statement) Print(args ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     args,
		multi:     false,
		name:      "print",
		open:      "print(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// PrintFunc renders the print built-in function.
func PrintFunc(f func(*Group)) *Statement {
	return newStatement().PrintFunc(f)
}

// PrintFunc renders the print built-in function.
func (g *Group) PrintFunc(f func(*Group)) *Statement {
	s := PrintFunc(f)
	g.items = append(g.items, s)
	return s
}

// PrintFunc renders the print built-in function.
func (s *Statement) PrintFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ")",
		multi:     false,
		name:      "print",
		open:      "print(",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Println renders the println built-in function.
func Println(args ...Code) *Statement {
	return newStatement().Println(args...)
}

// Println renders the println built-in function.
func (g *Group) Println(args ...Code) *Statement {
	s := Println(args...)
	g.items = append(g.items, s)
	return s
}

// Println renders the println built-in function.
func (s *Statement) Println(args ...Code) *Statement {
	g := &Group{
		close:     ")",
		items:     args,
		multi:     false,
		name:      "println",
		open:      "println(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// PrintlnFunc renders the println built-in function.
func PrintlnFunc(f func(*Group)) *Statement {
	return newStatement().PrintlnFunc(f)
}

// PrintlnFunc renders the println built-in function.
func (g *Group) PrintlnFunc(f func(*Group)) *Statement {
	s := PrintlnFunc(f)
	g.items = append(g.items, s)
	return s
}

// PrintlnFunc renders the println built-in function.
func (s *Statement) PrintlnFunc(f func(*Group)) *Statement {
	g := &Group{
		close:     ")",
		multi:     false,
		name:      "println",
		open:      "println(",
		separator: ",",
	}
	f(g)
	*s = append(*s, g)
	return s
}

// Real renders the real built-in function.
func Real(c Code) *Statement {
	return newStatement().Real(c)
}

// Real renders the real built-in function.
func (g *Group) Real(c Code) *Statement {
	s := Real(c)
	g.items = append(g.items, s)
	return s
}

// Real renders the real built-in function.
func (s *Statement) Real(c Code) *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{c},
		multi:     false,
		name:      "real",
		open:      "real(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Recover renders the recover built-in function.
func Recover() *Statement {
	return newStatement().Recover()
}

// Recover renders the recover built-in function.
func (g *Group) Recover() *Statement {
	s := Recover()
	g.items = append(g.items, s)
	return s
}

// Recover renders the recover built-in function.
func (s *Statement) Recover() *Statement {
	g := &Group{
		close:     ")",
		items:     []Code{},
		multi:     false,
		name:      "recover",
		open:      "recover(",
		separator: ",",
	}
	*s = append(*s, g)
	return s
}

// Number renders the number identifier.
func Number() *Statement {
	return newStatement().Number()
}

// Number renders the number identifier.
func (g *Group) Number() *Statement {
	s := Number()
	g.items = append(g.items, s)
	return s
}

// Number renders the number identifier.
func (s *Statement) Number() *Statement {
	t := token{
		content: "number",
		typ:     identifierToken,
	}
	*s = append(*s, t)
	return s
}

// String renders the string identifier.
func String() *Statement {
	return newStatement().String()
}

// String renders the string identifier.
func (g *Group) String() *Statement {
	s := String()
	g.items = append(g.items, s)
	return s
}

// String renders the string identifier.
func (s *Statement) String() *Statement {
	t := token{
		content: "string",
		typ:     identifierToken,
	}
	*s = append(*s, t)
	return s
}

// Boolean renders the boolean identifier.
func Boolean() *Statement {
	return newStatement().Boolean()
}

// Boolean renders the boolean identifier.
func (g *Group) Boolean() *Statement {
	s := Boolean()
	g.items = append(g.items, s)
	return s
}

// Boolean renders the boolean identifier.
func (s *Statement) Boolean() *Statement {
	t := token{
		content: "boolean",
		typ:     identifierToken,
	}
	*s = append(*s, t)
	return s
}

// Void renders the void identifier.
func Void() *Statement {
	return newStatement().Void()
}

// Void renders the void identifier.
func (g *Group) Void() *Statement {
	s := Void()
	g.items = append(g.items, s)
	return s
}

// Void renders the void identifier.
func (s *Statement) Void() *Statement {
	t := token{
		content: "void",
		typ:     identifierToken,
	}
	*s = append(*s, t)
	return s
}

// Undefined renders the undefined identifier.
func Undefined() *Statement {
	return newStatement().Undefined()
}

// Undefined renders the undefined identifier.
func (g *Group) Undefined() *Statement {
	s := Undefined()
	g.items = append(g.items, s)
	return s
}

// Undefined renders the undefined identifier.
func (s *Statement) Undefined() *Statement {
	t := token{
		content: "undefined",
		typ:     identifierToken,
	}
	*s = append(*s, t)
	return s
}

// Break renders the break keyword.
func Break() *Statement {
	return newStatement().Break()
}

// Break renders the break keyword.
func (g *Group) Break() *Statement {
	s := Break()
	g.items = append(g.items, s)
	return s
}

// Break renders the break keyword.
func (s *Statement) Break() *Statement {
	t := token{
		content: "break",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Throw renders the throw keyword.
func Throw() *Statement {
	return newStatement().Throw()
}

// Throw renders the throw keyword.
func (g *Group) Throw() *Statement {
	s := Throw()
	g.items = append(g.items, s)
	return s
}

// Throw renders the throw keyword.
func (s *Statement) Throw() *Statement {
	t := token{
		content: "throw",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Else renders the else keyword.
func Else() *Statement {
	return newStatement().Else()
}

// Else renders the else keyword.
func (g *Group) Else() *Statement {
	s := Else()
	g.items = append(g.items, s)
	return s
}

// Else renders the else keyword.
func (s *Statement) Else() *Statement {
	t := token{
		content: "else",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Var renders the var keyword.
func Var() *Statement {
	return newStatement().Var()
}

// Var renders the var keyword.
func (g *Group) Var() *Statement {
	s := Var()
	g.items = append(g.items, s)
	return s
}

// Var renders the var keyword.
func (s *Statement) Var() *Statement {
	t := token{
		content: "var",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Get renders the get keyword.
func Get() *Statement {
	return newStatement().Get()
}

// Get renders the get keyword.
func (g *Group) Get() *Statement {
	s := Get()
	g.items = append(g.items, s)
	return s
}

// Get renders the get keyword.
func (s *Statement) Get() *Statement {
	t := token{
		content: "get",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Module renders the module keyword.
func Module() *Statement {
	return newStatement().Module()
}

// Module renders the module keyword.
func (g *Group) Module() *Statement {
	s := Module()
	g.items = append(g.items, s)
	return s
}

// Module renders the module keyword.
func (s *Statement) Module() *Statement {
	t := token{
		content: "module",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Type renders the type keyword.
func Type() *Statement {
	return newStatement().Type()
}

// Type renders the type keyword.
func (g *Group) Type() *Statement {
	s := Type()
	g.items = append(g.items, s)
	return s
}

// Type renders the type keyword.
func (s *Statement) Type() *Statement {
	t := token{
		content: "type",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Instanceof renders the instanceof keyword.
func Instanceof() *Statement {
	return newStatement().Instanceof()
}

// Instanceof renders the instanceof keyword.
func (g *Group) Instanceof() *Statement {
	s := Instanceof()
	g.items = append(g.items, s)
	return s
}

// Instanceof renders the instanceof keyword.
func (s *Statement) Instanceof() *Statement {
	t := token{
		content: "instanceof",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Typeof renders the typeof keyword.
func Typeof() *Statement {
	return newStatement().Typeof()
}

// Typeof renders the typeof keyword.
func (g *Group) Typeof() *Statement {
	s := Typeof()
	g.items = append(g.items, s)
	return s
}

// Typeof renders the typeof keyword.
func (s *Statement) Typeof() *Statement {
	t := token{
		content: "typeof",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Public renders the public keyword.
func Public() *Statement {
	return newStatement().Public()
}

// Public renders the public keyword.
func (g *Group) Public() *Statement {
	s := Public()
	g.items = append(g.items, s)
	return s
}

// Public renders the public keyword.
func (s *Statement) Public() *Statement {
	t := token{
		content: "public",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Private renders the private keyword.
func Private() *Statement {
	return newStatement().Private()
}

// Private renders the private keyword.
func (g *Group) Private() *Statement {
	s := Private()
	g.items = append(g.items, s)
	return s
}

// Private renders the private keyword.
func (s *Statement) Private() *Statement {
	t := token{
		content: "private",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Enum renders the enum keyword.
func Enum() *Statement {
	return newStatement().Enum()
}

// Enum renders the enum keyword.
func (g *Group) Enum() *Statement {
	s := Enum()
	g.items = append(g.items, s)
	return s
}

// Enum renders the enum keyword.
func (s *Statement) Enum() *Statement {
	t := token{
		content: "enum",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Export renders the export keyword.
func Export() *Statement {
	return newStatement().Export()
}

// Export renders the export keyword.
func (g *Group) Export() *Statement {
	s := Export()
	g.items = append(g.items, s)
	return s
}

// Export renders the export keyword.
func (s *Statement) Export() *Statement {
	t := token{
		content: "export",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Finally renders the finally keyword.
func Finally() *Statement {
	return newStatement().Finally()
}

// Finally renders the finally keyword.
func (g *Group) Finally() *Statement {
	s := Finally()
	g.items = append(g.items, s)
	return s
}

// Finally renders the finally keyword.
func (s *Statement) Finally() *Statement {
	t := token{
		content: "finally",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// While renders the while keyword.
func While() *Statement {
	return newStatement().While()
}

// While renders the while keyword.
func (g *Group) While() *Statement {
	s := While()
	g.items = append(g.items, s)
	return s
}

// While renders the while keyword.
func (s *Statement) While() *Statement {
	t := token{
		content: "while",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Super renders the super keyword.
func Super() *Statement {
	return newStatement().Super()
}

// Super renders the super keyword.
func (g *Group) Super() *Statement {
	s := Super()
	g.items = append(g.items, s)
	return s
}

// Super renders the super keyword.
func (s *Statement) Super() *Statement {
	t := token{
		content: "super",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// This renders the this keyword.
func This() *Statement {
	return newStatement().This()
}

// This renders the this keyword.
func (g *Group) This() *Statement {
	s := This()
	g.items = append(g.items, s)
	return s
}

// This renders the this keyword.
func (s *Statement) This() *Statement {
	t := token{
		content: "this",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// In renders the in keyword.
func In() *Statement {
	return newStatement().In()
}

// In renders the in keyword.
func (g *Group) In() *Statement {
	s := In()
	g.items = append(g.items, s)
	return s
}

// In renders the in keyword.
func (s *Statement) In() *Statement {
	t := token{
		content: "in",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// True renders the true keyword.
func True() *Statement {
	return newStatement().True()
}

// True renders the true keyword.
func (g *Group) True() *Statement {
	s := True()
	g.items = append(g.items, s)
	return s
}

// True renders the true keyword.
func (s *Statement) True() *Statement {
	t := token{
		content: "true",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// False renders the false keyword.
func False() *Statement {
	return newStatement().False()
}

// False renders the false keyword.
func (g *Group) False() *Statement {
	s := False()
	g.items = append(g.items, s)
	return s
}

// False renders the false keyword.
func (s *Statement) False() *Statement {
	t := token{
		content: "false",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Any renders the any keyword.
func Any() *Statement {
	return newStatement().Any()
}

// Any renders the any keyword.
func (g *Group) Any() *Statement {
	s := Any()
	g.items = append(g.items, s)
	return s
}

// Any renders the any keyword.
func (s *Statement) Any() *Statement {
	t := token{
		content: "any",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Extends renders the extends keyword.
func Extends() *Statement {
	return newStatement().Extends()
}

// Extends renders the extends keyword.
func (g *Group) Extends() *Statement {
	s := Extends()
	g.items = append(g.items, s)
	return s
}

// Extends renders the extends keyword.
func (s *Statement) Extends() *Statement {
	t := token{
		content: "extends",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Static renders the static keyword.
func Static() *Statement {
	return newStatement().Static()
}

// Static renders the static keyword.
func (g *Group) Static() *Statement {
	s := Static()
	g.items = append(g.items, s)
	return s
}

// Static renders the static keyword.
func (s *Statement) Static() *Statement {
	t := token{
		content: "static",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Let renders the let keyword.
func Let() *Statement {
	return newStatement().Let()
}

// Let renders the let keyword.
func (g *Group) Let() *Statement {
	s := Let()
	g.items = append(g.items, s)
	return s
}

// Let renders the let keyword.
func (s *Statement) Let() *Statement {
	t := token{
		content: "let",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Package renders the package keyword.
func Package() *Statement {
	return newStatement().Package()
}

// Package renders the package keyword.
func (g *Group) Package() *Statement {
	s := Package()
	g.items = append(g.items, s)
	return s
}

// Package renders the package keyword.
func (s *Statement) Package() *Statement {
	t := token{
		content: "package",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Implements renders the implements keyword.
func Implements() *Statement {
	return newStatement().Implements()
}

// Implements renders the implements keyword.
func (g *Group) Implements() *Statement {
	s := Implements()
	g.items = append(g.items, s)
	return s
}

// Implements renders the implements keyword.
func (s *Statement) Implements() *Statement {
	t := token{
		content: "implements",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Function renders the function keyword.
func Function() *Statement {
	return newStatement().Function()
}

// Function renders the function keyword.
func (g *Group) Function() *Statement {
	s := Function()
	g.items = append(g.items, s)
	return s
}

// Function renders the function keyword.
func (s *Statement) Function() *Statement {
	t := token{
		content: "function",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Try renders the try keyword.
func Try() *Statement {
	return newStatement().Try()
}

// Try renders the try keyword.
func (g *Group) Try() *Statement {
	s := Try()
	g.items = append(g.items, s)
	return s
}

// Try renders the try keyword.
func (s *Statement) Try() *Statement {
	t := token{
		content: "try",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Yield renders the yield keyword.
func Yield() *Statement {
	return newStatement().Yield()
}

// Yield renders the yield keyword.
func (g *Group) Yield() *Statement {
	s := Yield()
	g.items = append(g.items, s)
	return s
}

// Yield renders the yield keyword.
func (s *Statement) Yield() *Statement {
	t := token{
		content: "yield",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Const renders the const keyword.
func Const() *Statement {
	return newStatement().Const()
}

// Const renders the const keyword.
func (g *Group) Const() *Statement {
	s := Const()
	g.items = append(g.items, s)
	return s
}

// Const renders the const keyword.
func (s *Statement) Const() *Statement {
	t := token{
		content: "const",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Continue renders the continue keyword.
func Continue() *Statement {
	return newStatement().Continue()
}

// Continue renders the continue keyword.
func (g *Group) Continue() *Statement {
	s := Continue()
	g.items = append(g.items, s)
	return s
}

// Continue renders the continue keyword.
func (s *Statement) Continue() *Statement {
	t := token{
		content: "continue",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Do renders the do keyword.
func Do() *Statement {
	return newStatement().Do()
}

// Do renders the do keyword.
func (g *Group) Do() *Statement {
	s := Do()
	g.items = append(g.items, s)
	return s
}

// Do renders the do keyword.
func (s *Statement) Do() *Statement {
	t := token{
		content: "do",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}

// Catch renders the catch keyword.
func Catch() *Statement {
	return newStatement().Catch()
}

// Catch renders the catch keyword.
func (g *Group) Catch() *Statement {
	s := Catch()
	g.items = append(g.items, s)
	return s
}

// Catch renders the catch keyword.
func (s *Statement) Catch() *Statement {
	t := token{
		content: "catch",
		typ:     keywordToken,
	}
	*s = append(*s, t)
	return s
}
