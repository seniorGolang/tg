package astra

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/structtag"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

var (
	ErrCouldNotResolvePackage = errors.New("could not resolve package")
	ErrUnexpectedSpec         = errors.New("unexpected spec")
	ErrNotInGoPath            = errors.New("not in GOPATH")
	ErrGoPathIsEmpty          = errors.New("GOPATH is empty")
)

type Option uint

const (
	IgnoreComments Option = 1 << iota
	IgnoreStructs
	IgnoreInterfaces
	IgnoreFunctions
	IgnoreMethods
	IgnoreTypes
	IgnoreVariables
	IgnoreConstants
	AllowAnyImportAliases
)

func concatOptions(ops []Option) (o Option) {
	for i := range ops {
		o |= ops[i]
	}
	return
}

func (o Option) check(what Option) bool {
	return o&what == what
}

// Parses ast.File and return all top-level declarations.
func ParseAstFile(file *ast.File, options ...Option) (*types.File, error) {

	opt := concatOptions(options)
	f := &types.File{
		Base: types.Base{
			Name: file.Name.Name,
			Docs: parseComments(file.Doc, opt),
		},
	}
	err := parseTopLevelDeclarations(file.Decls, f, opt)
	if err != nil {
		return nil, err
	}
	err = linkMethodsToStructs(f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func linkMethodsToStructs(f *types.File) error {
	for i := range f.Methods {
		structure, err := findStructByMethod(f, &f.Methods[i])
		if err != nil {
			return err
		}
		if structure != nil {
			structure.Methods = append(structure.Methods, &f.Methods[i])
			continue
		}
		typee, err := findTypeByMethod(f, &f.Methods[i])
		if err != nil {
			return err
		}
		if typee != nil {
			typee.Methods = append(typee.Methods, &f.Methods[i])
			continue
		}
	}
	return nil
}

func parseComments(group *ast.CommentGroup, o Option) (comments []string) {
	if o.check(IgnoreComments) {
		return
	}
	if group == nil {
		return
	}
	for _, comment := range group.List {
		comments = append(comments, comment.Text)
	}
	return
}

func parseTopLevelDeclarations(decls []ast.Decl, file *types.File, opt Option) error {

	for i := range decls {
		err := parseDeclaration(decls[i], file, opt)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	packagesPathNameCache = map[string]string{}
	mx                    sync.Mutex
)

func constructAliasName(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	importPath := strings.Trim(spec.Path.Value, `"`)
	mx.Lock()
	defer mx.Unlock()
	name, ok := packagesPathNameCache[importPath]
	if ok {
		return name
	}
	for _, p := range []string{build.Default.GOROOT, "vendor", build.Default.GOPATH} {
		name = findPackageName(p, importPath)
		if name != "" {
			break
		}
	}
	if name == "" {
		name = constructAliasNameString(spec.Path.Value)
	}
	packagesPathNameCache[importPath] = name
	return name
}

var importAliasReplacer = strings.NewReplacer("-", "")

func constructAliasNameString(str string) string {
	name := path.Base(strings.Trim(str, `"`))
	name = importAliasReplacer.Replace(name)
	if types.BuiltinTypes[name] || types.BuiltinFunctions[name] {
		name = "_" + name
	}
	return name
}

func findPackageName(src, path string) string {
	for _, gopath := range strings.Split(src, ":") {
		pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Join(gopath, path), nil, parser.PackageClauseOnly)
		if err != nil {
			continue
		}
		for k := range pkgs {
			return k
		}
	}
	return ""
}

func parseDeclaration(decl ast.Decl, file *types.File, opt Option) error {

	switch d := decl.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.IMPORT:
			var imports []*types.Import

			for _, spec := range d.Specs {
				spec, ok := spec.(*ast.ImportSpec)
				if !ok {
					continue // if !ok then comment
				}
				alias := constructAliasName(spec)
				imp := &types.Import{
					Base: types.Base{
						Name: alias,
						Docs: parseCommentFromSources(opt, d.Doc, spec.Doc, spec.Comment),
					},
					Package: strings.Trim(spec.Path.Value, `"`),
				}

				imports = append(imports, imp)
			}
			file.Imports = append(file.Imports, imports...)
		case token.VAR:
			if opt.check(IgnoreVariables) {
				return nil
			}
			vars, err := parseVariables(d, file, opt)
			if err != nil {
				return fmt.Errorf("parse variables %d:%d error: %v", d.Lparen, d.Rparen, err)
			}
			file.Vars = append(file.Vars, vars...)
		case token.CONST:
			if opt.check(IgnoreConstants) {
				return nil
			}
			consts, err := parseConstant(d, file, opt)
			if err != nil {
				return fmt.Errorf("parse constants %d:%d error: %v", d.Lparen, d.Rparen, err)
			}
			file.Constants = append(file.Constants, consts...)
		case token.TYPE:
			for i := range d.Specs {
				typeSpec := d.Specs[i].(*ast.TypeSpec)
				switch t := typeSpec.Type.(type) {
				case *ast.InterfaceType:
					if opt.check(IgnoreInterfaces) {
						return nil
					}
					methods, embedded, err := parseInterfaceMethods(t, file, opt)
					if err != nil {
						return err
					}
					file.Interfaces = append(file.Interfaces, types.Interface{
						Base: types.Base{
							Name: typeSpec.Name.Name,
							Docs: parseCommentFromSources(opt, d.Doc, typeSpec.Doc, typeSpec.Comment),
						},
						Methods:    methods,
						Interfaces: embedded,
					})
				case *ast.StructType:
					if opt.check(IgnoreStructs) {
						return nil
					}
					strFields, err := parseStructFields(t, file, opt)
					if err != nil {
						return fmt.Errorf("%s: can't parse struct fields: %v", typeSpec.Name.Name, err)
					}
					file.Structures = append(file.Structures, types.Struct{
						Base: types.Base{
							Name: typeSpec.Name.Name,
							Docs: parseCommentFromSources(opt, d.Doc, typeSpec.Doc, typeSpec.Comment),
						},
						Fields: strFields,
					})
				default:
					if opt.check(IgnoreTypes) {
						return nil
					}
					newType, _, err := parseByType(typeSpec.Type, file, opt)
					if err != nil {
						return fmt.Errorf("%s: can't parse type: %v", typeSpec.Name.Name, err)
					}
					file.Types = append(file.Types, types.FileType{Base: types.Base{
						Name: typeSpec.Name.Name,
						Docs: parseCommentFromSources(opt, d.Doc, typeSpec.Doc, typeSpec.Comment),
					}, Type: newType})
				}
			}
		}
	case *ast.FuncDecl:
		if opt.check(IgnoreFunctions) && opt.check(IgnoreMethods) {
			return nil
		}
		fn := types.Function{
			Base: types.Base{
				Name: d.Name.Name,
				Docs: parseComments(d.Doc, opt),
			},
		}
		err := parseFuncParamsAndResults(d.Type, &fn, file, opt)
		if err != nil {
			return fmt.Errorf("parse func %s error: %v", fn.Name, err)
		}
		if d.Recv != nil {
			if opt.check(IgnoreMethods) {
				return nil
			}
			rec, err := parseReceiver(d.Recv, file, opt)
			if err != nil {
				return err
			}
			file.Methods = append(file.Methods, types.Method{
				Function: fn,
				Receiver: *rec,
			})
		} else {
			if opt.check(IgnoreFunctions) {
				return nil
			}
			file.Functions = append(file.Functions, fn)
		}
	}
	return nil
}

func parseReceiver(list *ast.FieldList, file *types.File, opt Option) (*types.Variable, error) {
	recv, err := parseParams(list, file, opt)
	if err != nil {
		return nil, err
	}
	if len(recv) != 0 {
		return &recv[0], nil
	}
	return nil, fmt.Errorf("reciever not found for %d:%d", list.Pos(), list.End())
}

func parseConstant(decl *ast.GenDecl, file *types.File, opt Option) (constants []types.Constant, err error) {

	iotaMark := false
	var valType types.Type
	var iotaList *types.Constant
	for i := range decl.Specs {
		spec := decl.Specs[i].(*ast.ValueSpec)
		if len(spec.Values) > 0 && len(spec.Values) != len(spec.Names) {
			return nil, fmt.Errorf("amount of variables and their values not same %d:%d", spec.Pos(), spec.End())
		}
		for idx, name := range spec.Names {
			variable := types.Constant{
				Base: types.Base{
					Name: name.Name,
					Docs: parseCommentFromSources(opt, decl.Doc, spec.Doc, spec.Comment),
				},
			}
			switch {
			case spec.Type != nil:
				valType, iotaMark, err = parseByType(spec.Type, file, opt)
				if err != nil {
					return nil, fmt.Errorf("can't parse type: %v", err)
				}
				if len(spec.Values) > idx {
					_, _, iotaMark, err = parseByValue(spec.Values[idx], file, opt)
					if err != nil {
						return nil, fmt.Errorf("can't parse type: %v", err)
					}
				}
			case iotaMark:
			case len(spec.Values) > idx:
				variable.Value, valType, iotaMark, err = parseByValue(spec.Values[idx], file, opt)
				if err != nil {
					return nil, fmt.Errorf("can't parse type: %v", err)
				}
			default:
				return nil, fmt.Errorf("can't parse type: %d:%d", spec.Pos(), spec.End())
			}
			if iotaMark {
				if iotaList == nil {
					iotaList = &variable
				}
				variable.Type = valType
				variable.Iota = iotaMark
				iotaList.Constants = append(iotaList.Constants, variable)
				continue
			} else {
				variable.Type = valType
				constants = append(constants, variable)
			}
		}
	}
	if iotaList != nil {
		constants = append(constants, *iotaList)
	}
	return
}

func parseVariables(decl *ast.GenDecl, file *types.File, opt Option) (vars []types.Variable, err error) {

	iotaMark := false
	for i := range decl.Specs {
		spec := decl.Specs[i].(*ast.ValueSpec)
		if len(spec.Values) > 0 && len(spec.Values) != len(spec.Names) {
			return nil, fmt.Errorf("amount of variables and their values not same %d:%d", spec.Pos(), spec.End())
		}
		for idx, name := range spec.Names {
			variable := types.Variable{
				Base: types.Base{
					Name: name.Name,
					Docs: parseCommentFromSources(opt, decl.Doc, spec.Doc, spec.Comment),
				},
			}
			var (
				valType types.Type
				err     error
			)
			switch {
			case spec.Type != nil:
				valType, iotaMark, err = parseByType(spec.Type, file, opt)
				if err != nil {
					return nil, fmt.Errorf("can't parse type: %v", err)
				}
			case iotaMark:
				valType = iotaType
			case len(spec.Values) > idx:
				_, valType, iotaMark, err = parseByValue(spec.Values[idx], file, opt)
				if err != nil {
					return nil, fmt.Errorf("can't parse type: %v", err)
				}
			default:
				return nil, fmt.Errorf("can't parse type: %d:%d", spec.Pos(), spec.End())
			}
			variable.Type = valType
			vars = append(vars, variable)
		}
	}
	return
}

var iotaType = types.TName{TypeName: "iota"}

func parseByType(spec interface{}, file *types.File, opt Option) (tt types.Type, im bool, err error) {

	switch t := spec.(type) {
	case *ast.Ident:
		if t.Name == "iota" {
			return iotaType, true, nil
		}
		return types.TName{TypeName: t.Name}, false, nil
	case *ast.SelectorExpr:
		im, err := findImportByAlias(file, t.X.(*ast.Ident).Name)
		if err != nil && !opt.check(AllowAnyImportAliases) {
			return nil, false, fmt.Errorf("%s: %v", t.Sel.Name, err)
		}
		if im == nil && !opt.check(AllowAnyImportAliases) {
			return nil, false, fmt.Errorf("wrong import %d:%d", t.Pos(), t.End())
		}
		return types.TImport{Import: im, Next: types.TName{TypeName: t.Sel.Name}}, false, nil
	case *ast.StarExpr:
		next, iotaMark, err := parseByType(t.X, file, opt)
		if err != nil {
			return nil, false, err
		}
		if _, ok := next.(types.TPointer); ok {
			return types.TPointer{
				Next:             next.(types.TPointer).NextType(),
				NumberOfPointers: 1 + next.(types.TPointer).NumberOfPointers,
			}, iotaMark, nil
		}
		return types.TPointer{Next: next, NumberOfPointers: 1}, iotaMark, nil
	case *ast.ArrayType:
		l := parseArrayLen(t)
		next, iotaMark, err := parseByType(t.Elt, file, opt)
		if err != nil {
			return nil, false, err
		}
		switch l {
		case -3, -2:
			return types.TArray{Next: next, IsSlice: true}, iotaMark, nil
		case -1:
			return types.TArray{Next: next, IsEllipsis: true}, iotaMark, nil
		default:
			return types.TArray{Next: next, ArrayLen: l}, iotaMark, nil
		}
	case *ast.MapType:
		key, _, err := parseByType(t.Key, file, opt)
		if err != nil {
			return nil, false, err
		}
		value, _, err := parseByType(t.Value, file, opt)
		if err != nil {
			return nil, false, err
		}
		return types.TMap{Key: key, Value: value}, false, nil
	case *ast.InterfaceType:
		methods, embedded, err := parseInterfaceMethods(t, file, opt)
		if err != nil {
			return nil, false, err
		}
		return types.TInterface{
			Interface: &types.Interface{
				Base:       types.Base{},
				Methods:    methods,
				Interfaces: embedded,
			},
		}, false, nil
	case *ast.Ellipsis:
		next, iotaMark, err := parseByType(t.Elt, file, opt)
		if err != nil {
			return nil, false, err
		}
		return types.TEllipsis{Next: next}, iotaMark, nil
	case *ast.ChanType:
		next, iotaMark, err := parseByType(t.Value, file, opt)
		if err != nil {
			return nil, false, err
		}
		return types.TChan{Next: next, Direction: int(t.Dir)}, iotaMark, nil
	case *ast.ParenExpr:
		return parseByType(t.X, file, opt)
	case *ast.BadExpr:
		return nil, false, fmt.Errorf("bad expression")
	case *ast.FuncType:
		tt, err := parseFunction(t, file, opt)
		return tt, false, err
	case *ast.StructType:
		strFields, err := parseStructFields(t, file, opt)
		if err != nil {
			return nil, false, fmt.Errorf("can't parse anonymus struct fields: %v", err)
		}
		return types.Struct{
			Fields: strFields,
		}, false, nil
	default:
		return nil, false, fmt.Errorf("%v: %T", ErrUnexpectedSpec, t)
	}
}

func parseArrayLen(t *ast.ArrayType) int {
	if t == nil {
		return -2
	}
	switch l := t.Len.(type) {
	case *ast.Ellipsis:
		return -1
	case *ast.BasicLit:
		if l.Kind == token.INT {
			x, _ := strconv.Atoi(l.Value)
			return x
		}
		return 0
	}
	return -3
}

// Fill provided types.Type for cases, when variable's value is provided.
func parseByValue(spec interface{}, file *types.File, opt Option) (value interface{}, tt types.Type, iotaMark bool, err error) {

	switch t := spec.(type) {
	case *ast.BasicLit:
		return t.Value, types.TName{TypeName: t.Kind.String()}, false, nil
	case *ast.CompositeLit:
		return parseByValue(t.Type, file, opt)
	case *ast.SelectorExpr:
		im, err := findImportByAlias(file, t.X.(*ast.Ident).Name)
		if err != nil && !opt.check(AllowAnyImportAliases) {
			return nil, nil, false, fmt.Errorf("%s: %v", t.Sel.Name, err)
		}
		if im == nil && !opt.check(AllowAnyImportAliases) {
			return nil, nil, false, fmt.Errorf("wrong import %d:%d", t.Pos(), t.End())
		}
		return nil, types.TImport{Import: im}, false, nil
	case *ast.FuncType:
		fn, err := parseFunction(t, file, opt)
		if err != nil {
			return nil, nil, false, err
		}
		return nil, fn, false, nil
	case *ast.BinaryExpr:
		return parseByValue(t.X, file, opt) // parse one in pair
	case *ast.Ident: // iota case
		tt, iotaMark, err = parseByType(t, file, opt)
		return
	default:
		return nil, nil, false, nil
	}
}

// Collects and returns all interface methods.
// https://golang.org/ref/spec#Interface_types
func parseInterfaceMethods(ifaceType *ast.InterfaceType, file *types.File, opt Option) ([]*types.Function, []types.Variable, error) {
	var (
		fns      []*types.Function
		embedded []types.Variable
	)
	if ifaceType.Methods != nil {
		for _, method := range ifaceType.Methods.List {
			switch method.Type.(type) {
			case *ast.FuncType:
				// Functions (methods)
				fn, err := parseFunctionDeclaration(method, file, opt)
				if err != nil {
					return nil, nil, err
				}
				fns = append(fns, fn)
			case *ast.Ident:
				// Embedded interfaces
				iface, _, err := parseByType(method.Type, file, opt)
				if err != nil {
					return nil, nil, err
				}
				v := types.Variable{
					Base: types.Base{
						Name: "", // Because we embed interface.
						Docs: parseCommentFromSources(opt, method.Doc, method.Comment),
					},
					Type: iface,
				}
				embedded = append(embedded, v)
			}
		}
	}
	return fns, embedded, nil
}

func parseFunctionDeclaration(funcField *ast.Field, file *types.File, opt Option) (*types.Function, error) {
	funcType := funcField.Type.(*ast.FuncType)
	fn, err := parseFunction(funcType, file, opt)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", funcField.Names[0].Name, err)
	}
	fn.Name = funcField.Names[0].Name
	fn.Docs = parseComments(funcField.Doc, opt)
	return fn, nil
}

func parseFunction(funcType *ast.FuncType, file *types.File, opt Option) (*types.Function, error) {
	var fn = &types.Function{}
	err := parseFuncParamsAndResults(funcType, fn, file, opt)
	if err != nil {
		return nil, err
	}
	return fn, nil
}

func parseFuncParamsAndResults(funcType *ast.FuncType, fn *types.Function, file *types.File, opt Option) error {
	args, err := parseParams(funcType.Params, file, opt)
	if err != nil {
		return fmt.Errorf("can't parse args: %v", err)
	}
	fn.Args = args
	results, err := parseParams(funcType.Results, file, opt)
	if err != nil {
		return fmt.Errorf("can't parse results: %v", err)
	}
	fn.Results = results
	return nil
}

// Collects and returns all args/results from function or fields from structure.
func parseParams(fields *ast.FieldList, file *types.File, opt Option) ([]types.Variable, error) {
	var vars []types.Variable
	if fields == nil {
		return vars, nil
	}
	for _, field := range fields.List {
		if field.Type == nil {
			return nil, fmt.Errorf("param's type is nil %d:%d", field.Pos(), field.End())
		}
		t, _, err := parseByType(field.Type, file, opt)
		if err != nil {
			return nil, fmt.Errorf("wrong type of %s: %v", strings.Join(namesOfIdents(field.Names), ","), err)
		}
		docs := parseCommentFromSources(opt, field.Doc, field.Comment)
		if len(field.Names) == 0 {
			vars = append(vars, types.Variable{
				Base: types.Base{
					Docs: docs,
				},
				Type: t,
			})
		} else {
			for _, name := range field.Names {
				vars = append(vars, types.Variable{
					Base: types.Base{
						Name: name.Name,
						Docs: docs,
					},
					Type: t,
				})
			}
		}
	}
	return vars, nil
}

func parseTags(lit *ast.BasicLit) (tags map[string][]string, raw string) {
	if lit == nil {
		return
	}
	raw = lit.Value
	str := strings.Trim(lit.Value, "`")
	t, err := structtag.Parse(str)
	if err != nil {
		return
	}
	tags = make(map[string][]string)
	for _, tag := range t.Tags() {
		tags[tag.Key] = append([]string{tag.Name}, tag.Options...)
	}
	return
}

func parseStructFields(s *ast.StructType, file *types.File, opt Option) ([]types.StructField, error) {

	fields, err := parseParams(s.Fields, file, opt)
	if err != nil {
		return nil, err
	}
	var strF = make([]types.StructField, 0, len(fields))
	for i, f := range fields {
		var tags *ast.BasicLit
		// Fill tags, if Tag field exist in ast
		if i < len(s.Fields.List) {
			tags = s.Fields.List[i].Tag
		}
		parsedTags, rawTags := parseTags(tags)
		strF = append(strF, types.StructField{
			Variable: f,
			Tags:     parsedTags,
			RawTags:  rawTags,
		})
	}
	return strF, nil
}

func findImportByAlias(file *types.File, alias string) (*types.Import, error) {
	for _, imp := range file.Imports {
		if imp.Name == alias {
			return imp, nil
		}
	}
	// try to find by last segment of package path
	for _, imp := range file.Imports {
		if alias == path.Base(imp.Package) {
			return imp, nil
		}
	}

	return nil, fmt.Errorf("%v: %s", ErrCouldNotResolvePackage, alias)
}

func findStructByMethod(file *types.File, method *types.Method) (*types.Struct, error) {
	recType := method.Receiver.Type
	if !IsCommonReceiver(recType) {
		return nil, fmt.Errorf("%s has not common reciever", method.String())
	}
	name := types.TypeName(recType)
	if name == nil {
		return nil, nil
	}
	for i := range file.Structures {
		if file.Structures[i].Name == *name {
			return &file.Structures[i], nil
		}
	}
	return nil, nil
}

func findTypeByMethod(file *types.File, method *types.Method) (*types.FileType, error) {
	recType := method.Receiver.Type
	if !IsCommonReceiver(recType) {
		return nil, fmt.Errorf("%s has not common reciever", method.String())
	}
	name := types.TypeName(recType)
	if name == nil {
		return nil, nil
	}
	for i := range file.Types {
		if file.Types[i].Name == *name {
			return &file.Types[i], nil
		}
	}
	return nil, nil
}

func IsCommonReceiver(t types.Type) bool {
	for tt := t; tt != nil; {
		switch x := tt.(type) {
		case types.TArray, types.TInterface, types.TMap, types.TImport, types.Function:
			return false
		case types.TPointer:
			if x.NumberOfPointers > 1 {
				return false
			}
			tt = x.NextType()
		default:
			line, ok := tt.(types.LinearType)
			if !ok {
				return false
			}
			tt = line.NextType()
			continue
		}
	}
	return true
}
