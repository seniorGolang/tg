package astra

import (
	"fmt"
	"go/ast"
	astParser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

// Opens and parses file by name and return information about it.
func ParseFile(filename string, options ...Option) (*types.File, error) {

	path, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("can not filepath.Abs: %v", err)
	}
	fSet := token.NewFileSet()
	tree, err := astParser.ParseFile(fSet, path, nil, astParser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error when parse file: %v", err)
	}
	info, err := ParseAstFile(tree, options...)
	if err != nil {
		return nil, fmt.Errorf("error when parsing info from file: %v", err)
	}
	return info, nil
}

// Merges parsed files to one. Helpful, when you need full information about package.
func MergeFiles(files []*types.File) (*types.File, error) {

	targetFile := &types.File{}
	for _, file := range files {
		if file == nil {
			continue
		}
		// do not merge documentation.
		targetFile.Name = file.Name
		targetFile.Imports = mergeImports(targetFile.Imports, file.Imports)
		targetFile.Constants = append(targetFile.Constants, file.Constants...)
		targetFile.Vars = append(targetFile.Vars, file.Vars...)
		targetFile.Interfaces = append(targetFile.Interfaces, file.Interfaces...)
		targetFile.Structures = append(targetFile.Structures, file.Structures...)
		targetFile.Methods = append(targetFile.Methods, file.Methods...)
		targetFile.Types = append(targetFile.Types, file.Types...)
		targetFile.Functions = append(targetFile.Functions, file.Functions...)
	}
	err := linkMethodsToStructs(targetFile)
	if err != nil {
		return nil, err
	}
	return targetFile, nil
}

// Parses all .go files from directory.
// Deprecated: use GetPackage instead
func ParsePackage(path string, options ...Option) ([]*types.File, error) {

	p, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("can not filepath.Abs: %v", err)
	}
	files, err := os.ReadDir(p)
	if err != nil {
		return nil, fmt.Errorf("can not read dir: %v", err)
	}
	var parsedFiles = make([]*types.File, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		f, err := ParseFile(p+"/"+file.Name(), options...)
		if err != nil {
			return nil, fmt.Errorf("can not parse %s: %v", file.Name(), err)
		}
		parsedFiles = append(parsedFiles, f)
	}
	return parsedFiles, nil
}

func GetPackage(path string, options ...Option) (*types.File, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("can not filepath.Abs: %v", err)
	}
	fset := token.NewFileSet()
	pkgs, err := astParser.ParseDir(fset, p, nil, astParser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("can not parse dir: %v", err)
	}
	if len(pkgs) > 1 {
		return nil, fmt.Errorf("unexpected number of packages: expect 1, found %d", len(pkgs))
	}
	for _, pkg := range pkgs {
		f := ast.MergePackageFiles(pkg, ast.FilterUnassociatedComments|ast.FilterFuncDuplicates|ast.FilterImportDuplicates)
		return ParseAstFile(f, options...)
	}
	return nil, fmt.Errorf("unexpected number of packages: expect 1, found 0")
}

func ResolvePackagePath(outPath string) (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return "", ErrGoPathIsEmpty
	}

	absOutPath, err := filepath.Abs(filepath.Dir(outPath))
	if err != nil {
		return "", err
	}
	for _, path := range strings.Split(gopath, ":") {
		gopathSrc := filepath.Join(path, "src")
		if !strings.HasPrefix(absOutPath, gopathSrc) {
			continue
		}
		return absOutPath[len(gopathSrc)+1:], nil
	}
	return "", ErrNotInGoPath
}

func namesOfIdents(idents []*ast.Ident) (res []string) {
	for i := range idents {
		if idents[i] != nil {
			res = append(res, idents[i].Name)
		}
	}
	return
}

func mergeStringSlices(slices ...[]string) []string {
	if len(slices) == 0 {
		return nil
	}
	return append(slices[0], mergeStringSlices(slices[1:]...)...)
}

func parseCommentFromSources(opt Option, groups ...*ast.CommentGroup) []string {
	temp := make([][]string, len(groups))
	for i := range groups {
		temp[i] = parseComments(groups[i], opt)
	}
	return mergeStringSlices(temp...)
}
