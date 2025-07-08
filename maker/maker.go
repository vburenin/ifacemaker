package maker

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/imports"
)

// Method describes the code and documentation
// tied into a method
type Method struct {
	Name string
	Code string
	Docs []string
}

// declaredType identifies the name and package of a type declaration.
type declaredType struct {
	Name    string
	Package string
}

// Fullname returns a scoped Package.Name string out of this declaredType.
func (dt declaredType) Fullname() string {
	return fmt.Sprintf("%s.%s", dt.Package, dt.Name)
}

// Lines return a []string consisting of
// the documentation and code appended
// in chronological order
func (m *Method) Lines() []string {
	var lines []string
	lines = append(lines, m.Docs...)
	lines = append(lines, m.Code)
	return lines
}

// GetTypeDeclarationName extract the name of the type of this declaration if it refers to a type declaration.
// Otherwise, it returns an empty string.
func GetTypeDeclarationName(decl ast.Decl) string {
	gd, ok := decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.TYPE || len(gd.Specs) == 0 {
		return ""
	}

	if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
		return ts.Name.Name
	}

	return ""
}

// getTypeDeclarationNames extracts all type names from the given declaration.
// If the declaration is not a type declaration, it returns nil.
func getTypeDeclarationNames(decl ast.Decl) []string {
	gd, ok := decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.TYPE {
		return nil
	}

	var names []string
	for _, spec := range gd.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			names = append(names, ts.Name.Name)
		}
	}
	return names
}

// GetReceiverTypeName takes in the entire
// source code and a single declaration.
// It then checks if the declaration is a
// function declaration, if it is, it uses
// the GetReceiverType to check whether
// the declaration is a method or a function
// if it is a function we fatally stop.
// If it is a method we retrieve the type
// of the receiver based on the types
// start and end pos in combination with
// the actual source code.
// It then returns the name of the
// receiver type and the function declaration
//
// Behavior is undefined for a src []byte that
// isn't the source of the possible FuncDecl fl
func GetReceiverTypeName(src []byte, fl ast.Decl) (string, *ast.FuncDecl) {
	fd, ok := fl.(*ast.FuncDecl)
	if !ok {
		return "", nil
	}
	t, err := GetReceiverType(fd)
	if err != nil {
		return "", nil
	}
	st := string(src[t.Pos()-1 : t.End()-1])
	if len(st) > 0 && st[0] == '*' {
		st = st[1:]
	}
	// Strip generic type parameters if present, e.g. Foo[T] -> Foo
	if m := regexp.MustCompile(`^(\w+)(?:\[.+\])?$`).FindStringSubmatch(st); m != nil {
		st = m[1]
	}
	return st, fd
}

// GetReceiverType checks if the FuncDecl
// is a function or a method. If it is a
// function it returns a nil ast.Expr and
// a non-nil err. If it is a method it uses
// a hardcoded 0 index to fetch the receiver
// because a method can only have 1 receiver.
// Which can make you wonder why it is a
// list in the first place, but this type
// from the `ast` pkg is used in other
// places than for receivers
func GetReceiverType(fd *ast.FuncDecl) (ast.Expr, error) {
	if fd.Recv == nil {
		return nil, fmt.Errorf("fd is not a method, it is a function")
	}
	return fd.Recv.List[0].Type, nil
}

// getEmbeddedStructName returns the base struct name for an embedded field.
// It unwraps pointers and generic instantiations to get the underlying type
// identifier so embedded methods can be associated correctly.
func getEmbeddedStructName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return getEmbeddedStructName(e.X)
	case *ast.IndexExpr:
		return getEmbeddedStructName(e.X)
	case *ast.IndexListExpr:
		return getEmbeddedStructName(e.X)
	default:
		log.Printf("Unsupported ast.Expr type: %T", expr)
		return ""
	}
}

// reMatchTypename matches any of the following to extract the <type>:
//
//	*<type>
//	[]<type>
//	[]*<type>
//	map[<keyType>]<type>
//	map[<keyType>]*<type>
//
// Updated regex to support generic type parameters like Foo[T any].
// The prefix (e.g. pointers or collection modifiers) is optional so that
// generic types without any modifiers are also matched correctly. The first
// capture group contains the prefix, if any, and the second group contains the
// base type name.
var reMatchTypename = regexp.MustCompile(`^((\[\]|\*|map\[[^\]]+\])*)?(\w+)(\[.+\])?$`)

// baseIdentName returns the base identifier name from a type expression. It
// recursively follows pointer, slice, map and generic expressions until the
// underlying identifier is reached. If no identifier is present, an empty
// string is returned.
func baseIdentName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return baseIdentName(e.X)
	case *ast.ArrayType:
		return baseIdentName(e.Elt)
	case *ast.MapType:
		return baseIdentName(e.Value)
	case *ast.IndexExpr:
		return baseIdentName(e.X)
	case *ast.IndexListExpr:
		return baseIdentName(e.X)
	case *ast.Ellipsis:
		return baseIdentName(e.Elt)
	default:
		return ""
	}
}

// FormatFieldList takes in the source code as a []byte and a FuncDecl
// parameters or return values as a FieldList. It returns a slice of strings
// where each element is one parameter or return value formatted as it appears
// in the source. If the FieldList input is nil, it returns nil.
func FormatFieldList(src []byte, fl *ast.FieldList, pkgName string, declaredTypes []declaredType) []string {
	if fl == nil {
		return nil
	}
	var parts []string
	for _, l := range fl.List {
		names := make([]string, len(l.Names))
		for i, n := range l.Names {
			names[i] = n.Name
		}
		t := string(src[l.Type.Pos()-1 : l.Type.End()-1])
		// Try to match <modifier><type>. If matched variable `match` will look like this for t=="[]Category":
		// match[0][0] = "[]Category"
		// match[0][1] = "[]"
		// match[0][2] = "Category"
		t2 := baseIdentName(l.Type)
		match := reMatchTypename.FindStringSubmatch(t)
		var prefix, generics string
		if match != nil {
			// Extract prefix (e.g. *[] or map[]) and generic parameters
			prefix = match[1]
			generics = match[3]
		}

		for _, dt := range declaredTypes {
			if t2 == dt.Name && pkgName != dt.Package {
				if match != nil {
					t = prefix + dt.Fullname() + generics
				} else {
					t = strings.Replace(t, t2, dt.Fullname(), 1)
				}
				break
			}
		}

		// Strip destination package prefix when source code imports the
		// same package we are generating into. This handles pointers,
		// arrays, maps and other composite types.
		regexString := fmt.Sprintf(`(^|[^\w])%s\.`, regexp.QuoteMeta(pkgName))
		t = regexp.MustCompile(regexString).ReplaceAllString(t, "$1")

		if len(names) > 0 {
			typeSharingArgs := strings.Join(names, ", ")
			parts = append(parts, fmt.Sprintf("%s %s", typeSharingArgs, t))
		} else {
			parts = append(parts, t)
		}
	}
	return parts
}

// FormatCode sets the options of the imports
// pkg and then applies the Process method
// which by default removes all of the imports
// not used and formats the remaining docs,
// imports and code like `gofmt`. It will
// e.g. remove paranthesis around a unnamed
// single return type
func FormatCode(code string) ([]byte, error) {
	opts := &imports.Options{
		TabIndent: true,
		TabWidth:  2,
		Fragment:  true,
		Comments:  true,
	}
	return imports.Process("", []byte(code), opts)
}

// MakeInterface takes in all of the items
// required for generating the interface,
// it then simply concatenates them all
// to an array, joins this array to a string
// with newline and passes it on to FormatCode
// which then directly returns the result
func MakeInterface(comment, pkgName, ifaceName, ifaceComment, typeParams string, methods []string, imports []string) ([]byte, error) {
	output := []string{
		"// " + comment,
		"",
		"package " + pkgName,
		"import (",
	}
	output = append(output, imports...)
	output = append(output,
		")",
		"",
	)
	if len(ifaceComment) > 0 {
		prefix := "// "
		if strings.HasPrefix(ifaceComment, "go:generate") {
			prefix = "//"
		}
		output = append(output, fmt.Sprintf("%s%s", prefix, strings.ReplaceAll(ifaceComment, "\n", "\n// ")))
	}
	output = append(output, fmt.Sprintf("type %s%s interface {", ifaceName, typeParams))
	output = append(output, methods...)
	output = append(output, "}")
	code := strings.Join(output, "\n")
	return FormatCode(code)
}

// ParseDeclaredTypes inspect given src code to find type declaractions.
func ParseDeclaredTypes(src []byte) (declaredTypes []declaredType) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err.Error())
	}

	sourcePackageName := a.Name.Name

	for _, d := range a.Decls {
		for _, name := range getTypeDeclarationNames(d) {
			declaredTypes = append(declaredTypes, declaredType{
				Name:    name,
				Package: sourcePackageName,
			})
		}
	}

	return
}

// ParseEmbeddingGraph inspects the given source code to find
// the embedding relationship between structs
func ParseEmbeddingGraph(src []byte) map[string][]string {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "src.go", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Track the embedding graph
	embeddingGraph := make(map[string][]string)
	for _, decl := range file.Decls {
		// Skip non-type declaration
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			// Skip non-type specifications
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Skip non-struct types
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Process struct types
			childStructName := typeSpec.Name.Name
			if _, ok := embeddingGraph[childStructName]; !ok {
				embeddingGraph[childStructName] = []string{}
			}
			for _, fieldType := range structType.Fields.List {
				// Skip non-embedded fields
				if len(fieldType.Names) > 0 {
					continue
				}

				name := getEmbeddedStructName(fieldType.Type)
				if name != "" {
					embeddingGraph[childStructName] = append(embeddingGraph[childStructName], name)
				}
			}
		}
	}

	return embeddingGraph
}

// ParseStruct takes in a piece of source code as a
// []byte, the name of the struct it should base the
// interface on and a bool saying whether it should
// include docs.  It then returns an []Method where
// Method contains the method declaration(not the code)
// that is required for the interface and any documentation
// if included.
// It also returns a []string containing all of the imports
// including their aliases regardless of them being used or
// not, the imports not used will be removed later using the
// 'imports' pkg If anything goes wrong, this method will
// fatally stop the execution
func ParseStruct(src []byte, structName string, copyDocs bool, copyTypeDocs bool, pkgName string, declaredTypes []declaredType, importModule string, withNotExported bool, embeddedStructNamesSet map[string]struct{}, withPromoted bool) (methods []Method, imports []string, typeDoc string, typeParams string) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Extract type parameters for the struct if present.
	for _, decl := range a.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}
			if ts.TypeParams != nil {
				// Subtracting 1 from ts.TypeParams.Pos() and ts.TypeParams.End() adjusts the indices
				// to match the zero-based indexing of the src slice. This ensures the correct
				// extraction of type parameters from the source code.
				typeParams = string(src[ts.TypeParams.Pos()-1 : ts.TypeParams.End()-1])
			}
		}
	}

	for _, i := range a.Imports {
		if i.Name != nil {
			imports = append(imports, fmt.Sprintf("%s %s", i.Name.String(), i.Path.Value))
		} else {
			imports = append(imports, i.Path.Value)
		}
	}

	if importModule != "" {
		imports = append(imports, fmt.Sprintf(". %s", strconv.Quote(importModule)))
	}

	// Track methods that are already processed. Keyed by method name
	// so that promoted methods overridden in the main struct are skipped.
	methodSet := make(map[string]struct{})

	// Process direct methods first
	for _, d := range a.Decls {
		if a, fd := GetReceiverTypeName(src, d); a == structName {
			mName := fd.Name.String()
			if _, ok := methodSet[mName]; ok {
				continue
			}
			if !withNotExported && !fd.Name.IsExported() {
				continue
			}
			params := FormatFieldList(src, fd.Type.Params, pkgName, declaredTypes)
			ret := FormatFieldList(src, fd.Type.Results, pkgName, declaredTypes)

			mName = fd.Name.String()
			method := ""
			if len(ret) == 0 {
				method = fmt.Sprintf("%s(%s)", mName, strings.Join(params, ", "))
			} else {
				method = fmt.Sprintf("%s(%s) (%s)", mName, strings.Join(params, ", "), strings.Join(ret, ", "))
			}

			var docs []string
			if fd.Doc != nil && copyDocs {
				for _, d := range fd.Doc.List {
					docs = append(docs, string(src[d.Pos()-1:d.End()-1]))
				}
			}
			methods = append(methods, Method{
				Name: mName,
				Code: method,
				Docs: docs,
			})
			methodSet[mName] = struct{}{}
		}
	}

	// Add promoted methods next
	if withPromoted {
		for _, d := range a.Decls {
			a, fd := GetReceiverTypeName(src, d)
			_, isEmbedded := embeddedStructNamesSet[a]
			if isEmbedded {
				mName := fd.Name.String()
				if _, ok := methodSet[mName]; ok {
					continue
				}
				if !withNotExported && !fd.Name.IsExported() {
					continue
				}
				params := FormatFieldList(src, fd.Type.Params, pkgName, declaredTypes)
				ret := FormatFieldList(src, fd.Type.Results, pkgName, declaredTypes)
				method := ""
				if len(ret) == 0 {
					method = fmt.Sprintf("%s(%s)", mName, strings.Join(params, ", "))
				} else {
					method = fmt.Sprintf("%s(%s) (%s)", mName, strings.Join(params, ", "), strings.Join(ret, ", "))
				}
				var docs []string
				if fd.Doc != nil && copyDocs {
					for _, d := range fd.Doc.List {
						docs = append(docs, string(src[d.Pos()-1:d.End()-1]))
					}
				}
				methods = append(methods, Method{
					Name: mName,
					Code: method,
					Docs: docs,
				})
				methodSet[mName] = struct{}{}
			}
		}
	}

	if copyTypeDocs {
		pkgDoc, err := doc.NewFromFiles(fset, []*ast.File{a}, "", doc.AllDecls)
		if err == nil {
			for _, t := range pkgDoc.Types {
				if t.Name == structName {
					typeDoc = strings.TrimSuffix(t.Doc, "\n")
				}
			}
		}
	}

	return
}

// MakeOptions contains options for the Make function.
type MakeOptions struct {
	Files           []string
	StructType      string
	Comment         string
	PkgName         string
	WithPromoted    bool
	IfaceName       string
	IfaceComment    string
	ImportModule    string
	CopyDocs        bool
	CopyTypeDoc     bool
	ExcludeMethods  []string
	WithNotExported bool
}

// validateStructType checks input struct type against the parsed declared
// types and returns true when present
func validateStructType(types []declaredType, stType string) bool {
	for _, v := range types {
		if v.Name == stType {
			return true
		}

	}
	return false

}

func Make(options MakeOptions) ([]byte, error) {
	var (
		allMethods       []string
		allImports       []string
		allDeclaredTypes []declaredType

		fullEmbeddingGraph = make(map[string][]string)
		mset               = make(map[string]struct{})
		iset               = make(map[string]struct{})
		tset               = make(map[string]struct{})
	)

	var (
		typeDoc     string
		ifaceParams string
	)

	// First pass on all files to find declared types
	for _, f := range options.Files {
		b, err := os.ReadFile(f)
		if err != nil {
			return []byte{}, err
		}
		types := ParseDeclaredTypes(b)
		graph := ParseEmbeddingGraph(b)

		// Track if we've seen the input Struct type
		for _, t := range types {
			if _, ok := tset[t.Fullname()]; !ok {
				allDeclaredTypes = append(allDeclaredTypes, t)
				tset[t.Fullname()] = struct{}{}
			}
		}

		// Track the full call graph
		for key, values := range graph {
			if _, ok := fullEmbeddingGraph[key]; !ok {
				fullEmbeddingGraph[key] = []string{}
			}
			fullEmbeddingGraph[key] = append(fullEmbeddingGraph[key], values...)
		}
	}

	// Validate at least one file contains the input struct Type
	if !validateStructType(allDeclaredTypes, options.StructType) {
		return []byte{},
			fmt.Errorf("%q structtype not found in input files",
				options.StructType)
	}

	excludedMethods := make(map[string]struct{}, len(options.ExcludeMethods))
	for _, mName := range options.ExcludeMethods {
		excludedMethods[mName] = struct{}{}
	}

	embeddedStructNamesSet := make(map[string]struct{})
	queue := []string{options.StructType}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, embeddedStruct := range fullEmbeddingGraph[curr] {
			if _, ok := embeddedStructNamesSet[embeddedStruct]; ok {
				continue
			}
			embeddedStructNamesSet[embeddedStruct] = struct{}{}
			queue = append(queue, embeddedStruct)
		}
	}

	// Second pass to build up the interface
	for _, f := range options.Files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		methods, imports, parsedTypeDoc, parsedParams := ParseStruct(src, options.StructType, options.CopyDocs, options.CopyTypeDoc, options.PkgName, allDeclaredTypes, options.ImportModule, options.WithNotExported, embeddedStructNamesSet, options.WithPromoted)
		for _, m := range methods {
			if _, ok := excludedMethods[m.Name]; ok {
				continue
			}

			// Use m.Name as the key to ensure uniqueness of methods in mset.
			if _, ok := mset[m.Name]; !ok {
				allMethods = append(allMethods, m.Lines()...)
				mset[m.Name] = struct{}{}
			}
		}
		for _, i := range imports {
			if _, ok := iset[i]; !ok {
				allImports = append(allImports, i)
				iset[i] = struct{}{}
			}
		}
		if typeDoc == "" {
			typeDoc = parsedTypeDoc
		}
		if ifaceParams == "" {
			ifaceParams = parsedParams
		}
	}

	if typeDoc != "" {
		options.IfaceComment = fmt.Sprintf("%s\n%s", options.IfaceComment, typeDoc)
	}

	result, err := MakeInterface(options.Comment, options.PkgName, options.IfaceName, options.IfaceComment, ifaceParams, allMethods, allImports)
	if err != nil {
		return nil, err
	}

	return result, nil
}
