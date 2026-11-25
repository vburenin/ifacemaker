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
	"slices"
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
	// Promoted is true for a method promoted from an embedded struct
	// rather than declared on the struct itself.
	Promoted bool

	// receiver is the name of the embedded struct a promoted method is
	// declared on.
	receiver string
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

// qualifyType returns the source text of the type expression expr, with every
// identifier that names a type declared in a package other than pkgName
// prefixed by that package's name. Types that are already qualified with a
// package name, and the names of parameters, results, struct fields and
// interface methods inside the expression, are left as they are.
func qualifyType(src []byte, expr ast.Expr, pkgName string, declaredTypes []declaredType) string {
	start, end := int(expr.Pos())-1, int(expr.End())-1
	var b strings.Builder
	names := make(map[*ast.Ident]struct{})
	ast.Inspect(expr, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			// Already qualified, e.g. pkg.Type.
			return false
		case *ast.Field:
			for _, name := range n.Names {
				names[name] = struct{}{}
			}
		case *ast.Ident:
			if _, ok := names[n]; ok {
				return false
			}
			for _, dt := range declaredTypes {
				if n.Name == dt.Name && pkgName != dt.Package {
					// ast.Inspect visits nodes in source order, so the
					// offsets of qualified identifiers only increase.
					offset := int(n.Pos()) - 1
					b.Write(src[start:offset])
					b.WriteString(dt.Package + ".")
					start = offset
					break
				}
			}
		}
		return true
	})
	b.Write(src[start:end])
	return b.String()
}

// FormatFieldList takes in the source code as a []byte and a FuncDecl
// parameters or return values as a FieldList. It returns a slice of strings
// where each element is one parameter or return value formatted as it appears
// in the source. Types declared in a package other than pkgName are qualified
// with their package name, and qualifiers naming pkgName itself are removed.
// If the FieldList input is nil, it returns nil.
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
		t := qualifyType(src, l.Type, pkgName, declaredTypes)

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

// reMatchDirective matches with any go directive
//
// `go:build tag`
// `go:generate command`
// `go:embed file`
//
//	...
var reMatchDirective = regexp.MustCompile(`^(//go|go):\S+`)

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
		if reMatchDirective.MatchString(ifaceComment) {
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

// methodScopeTypes returns the declared types that unqualified identifiers in
// the signature of fd, a method in a file of package srcPkg, may refer to. A
// type declared in srcPkg hides types of the same name declared in other
// packages, and the receiver's type parameters hide every declared type of
// the same name.
func methodScopeTypes(declaredTypes []declaredType, srcPkg string, fd *ast.FuncDecl) []declaredType {
	hidden := make(map[string]struct{})
	for _, dt := range declaredTypes {
		if dt.Package == srcPkg {
			hidden[dt.Name] = struct{}{}
		}
	}
	typeParams := make(map[string]struct{})
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		recv := fd.Recv.List[0].Type
		if star, ok := recv.(*ast.StarExpr); ok {
			recv = star.X
		}
		var indices []ast.Expr
		switch r := recv.(type) {
		case *ast.IndexExpr:
			indices = []ast.Expr{r.Index}
		case *ast.IndexListExpr:
			indices = r.Indices
		}
		for _, idx := range indices {
			if id, ok := idx.(*ast.Ident); ok {
				typeParams[id.Name] = struct{}{}
			}
		}
	}
	var scoped []declaredType
	for _, dt := range declaredTypes {
		if _, ok := typeParams[dt.Name]; ok {
			continue
		}
		if _, ok := hidden[dt.Name]; ok && dt.Package != srcPkg {
			continue
		}
		scoped = append(scoped, dt)
	}
	return scoped
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
	var all []Method
	all, imports, typeDoc, typeParams = parseStruct(src, structName, copyDocs, copyTypeDocs, pkgName, declaredTypes, importModule, withNotExported, embeddedStructNamesSet, withPromoted)
	// Keep the first method of each name: methods declared on the struct
	// come first and hide promoted methods of the same name.
	seen := make(map[string]struct{}, len(all))
	for _, m := range all {
		if _, ok := seen[m.Name]; !ok {
			seen[m.Name] = struct{}{}
			methods = append(methods, m)
		}
	}
	return methods, imports, typeDoc, typeParams
}

// parseStruct implements ParseStruct, but returns every method declared on an
// embedded struct, even when several share a name, so that Make can pick the
// one Go promotes.
func parseStruct(src []byte, structName string, copyDocs bool, copyTypeDocs bool, pkgName string, declaredTypes []declaredType, importModule string, withNotExported bool, embeddedStructNamesSet map[string]struct{}, withPromoted bool) (methods []Method, imports []string, typeDoc string, typeParams string) {
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

	// Track the methods declared on the struct, keyed by method name, so
	// that a method declared twice is only returned once.
	methodSet := make(map[string]struct{})
	srcPkg := a.Name.Name

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
			scoped := methodScopeTypes(declaredTypes, srcPkg, fd)
			params := FormatFieldList(src, fd.Type.Params, pkgName, scoped)
			ret := FormatFieldList(src, fd.Type.Results, pkgName, scoped)

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
					commentLine := string(src[d.Pos()-1 : d.End()-1])
					if !reMatchDirective.MatchString(commentLine) {
						docs = append(docs, commentLine)
					}
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
				if !withNotExported && !fd.Name.IsExported() {
					continue
				}
				scoped := methodScopeTypes(declaredTypes, srcPkg, fd)
				params := FormatFieldList(src, fd.Type.Params, pkgName, scoped)
				ret := FormatFieldList(src, fd.Type.Results, pkgName, scoped)
				method := ""
				if len(ret) == 0 {
					method = fmt.Sprintf("%s(%s)", mName, strings.Join(params, ", "))
				} else {
					method = fmt.Sprintf("%s(%s) (%s)", mName, strings.Join(params, ", "), strings.Join(ret, ", "))
				}
				var docs []string
				if fd.Doc != nil && copyDocs {
					for _, d := range fd.Doc.List {
						commentLine := string(src[d.Pos()-1 : d.End()-1])
						if !reMatchDirective.MatchString(commentLine) {
							docs = append(docs, commentLine)
						}
					}
				}
				methods = append(methods, Method{
					Name:     mName,
					Code:     method,
					Docs:     docs,
					Promoted: true,
					receiver: a,
				})
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

// embedding describes where an embedded struct is found in the struct an
// interface is generated for.
type embedding struct {
	// depth is the shallowest depth at which the struct is embedded,
	// starting at 1 for a struct embedded directly.
	depth int
	// multiple reports whether the struct is embedded through more than
	// one path at that depth.
	multiple bool
}

// embeddings returns the structs embedded, directly or indirectly, in
// structName according to graph. Like Go's selector lookup, it walks the
// graph one depth at a time and ignores a struct already found at a
// shallower depth.
func embeddings(graph map[string][]string, structName string) map[string]embedding {
	found := make(map[string]embedding)
	current := map[string]int{structName: 1} // struct name to number of paths
	for depth := 1; len(current) > 0; depth++ {
		next := make(map[string]int)
		for name, paths := range current {
			for _, e := range graph[name] {
				if _, ok := found[e]; !ok && e != structName {
					next[e] += paths
				}
			}
		}
		for name, paths := range next {
			found[name] = embedding{depth: depth, multiple: paths > 1}
		}
		current = next
	}
	return found
}

// selectPromoted returns, in their original order, the candidate methods that
// Go promotes: for each name, the method declared on the shallowest embedded
// struct. A name declared on more than one struct at that depth, or on a
// struct embedded through more than one path at that depth, is ambiguous and
// not promoted at all.
func selectPromoted(candidates []Method, embedded map[string]embedding) []Method {
	type selection struct {
		receiver  string
		depth     int
		ambiguous bool
	}
	selected := make(map[string]*selection)
	for _, m := range candidates {
		e := embedded[m.receiver]
		s, ok := selected[m.Name]
		switch {
		case !ok || e.depth < s.depth:
			selected[m.Name] = &selection{receiver: m.receiver, depth: e.depth, ambiguous: e.multiple}
		case e.depth == s.depth && m.receiver != s.receiver:
			s.ambiguous = true
		}
	}

	var methods []Method
	for _, m := range candidates {
		if s := selected[m.Name]; !s.ambiguous && m.receiver == s.receiver {
			methods = append(methods, m)
		}
	}
	return methods
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
		// The Parse* helpers exit the process on a syntax error and do not
		// know the file name, so report syntax errors from here instead.
		if _, err := parser.ParseFile(token.NewFileSet(), f, b, parser.ParseComments); err != nil {
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

		// Track the full call graph. A struct seen in several files, or a
		// file passed twice, must not make an embedding look duplicated.
		for key, values := range graph {
			if _, ok := fullEmbeddingGraph[key]; !ok {
				fullEmbeddingGraph[key] = []string{}
			}
			for _, v := range values {
				if !slices.Contains(fullEmbeddingGraph[key], v) {
					fullEmbeddingGraph[key] = append(fullEmbeddingGraph[key], v)
				}
			}
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

	embedded := embeddings(fullEmbeddingGraph, options.StructType)
	embeddedStructNamesSet := make(map[string]struct{}, len(embedded))
	for name := range embedded {
		embeddedStructNamesSet[name] = struct{}{}
	}

	// Second pass to build up the interface
	var directMethods, promotedMethods []Method
	for _, f := range options.Files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		methods, imports, parsedTypeDoc, parsedParams := parseStruct(src, options.StructType, options.CopyDocs, options.CopyTypeDoc, options.PkgName, allDeclaredTypes, options.ImportModule, options.WithNotExported, embeddedStructNamesSet, options.WithPromoted)
		for _, m := range methods {
			if m.Promoted {
				promotedMethods = append(promotedMethods, m)
			} else {
				directMethods = append(directMethods, m)
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

	// Methods declared on the struct itself take precedence over promoted
	// methods of the same name, whichever files they are declared in.
	for _, m := range append(directMethods, selectPromoted(promotedMethods, embedded)...) {
		if _, ok := excludedMethods[m.Name]; ok {
			continue
		}

		// Use m.Name as the key to ensure uniqueness of methods in mset.
		if _, ok := mset[m.Name]; !ok {
			allMethods = append(allMethods, m.Lines()...)
			mset[m.Name] = struct{}{}
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
