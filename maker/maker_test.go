package maker

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var (
	src = []byte(`
package main

import (
	notmain "fmt"
)

// Person ...
type Person struct {
	name      string
	age       int
	telephone string
	noPointer notmain.Formatter
	pointer   *notmain.Formatter
}

// Name ...
func (p *Person) Name() string {
	return p.name
}

// SetName ...
func (p *Person) SetName(name string) {
	p.name = name
}

// Age ...
func (p *Person) Age() int {
	return p.age
}

// Age ...
func (p *Person) SetAge(age int) {
	p.Age = age
}

// AgeAndName ...
func (p *Person) GetAgeAndName() {
	return p.age, p.name
}

func (p *Person) SetAgeAndName(name string, age int) {
	p.name = name
	p.age = age
}

func (p *Person) GetNameAndTelephone() (name, telephone string) {
	telephone = p.telephone
	name = p.name
	return
}

func (p *Person) SetNameAndTelephone(name, telephone string) {
	p.name = name
	p.telephone = telephone
}

func (p *Person) ReturnPointer() *notmain.Formatter {
	return nil
}

func (p *Person) ReturnNoPointer() notmain.Formatter {
	return nil
}

func (p *Person) ArgumentNoPointer(formatter notmain.Formatter) {

}

func (p *Person) ArgumentPointer(formatter *notmain.Formatter) {

}

func (p *Person) SecondArgumentPointer(abc int, formatter *notmain.Formatter) {

}

func (p *Person) unexportedFuncOne() bool {
	return true
}

func (p *Person) unexportedFuncTwo() string {
	return "hi!"
}
func SomeFunction() string {
	return "Something"
}

type SomeType struct{}

// Turing is a person
type Turing struct {
	*Person
}`)
)

func TestLines(t *testing.T) {
	docs := []string{`// TestMethod is great`}
	code := `func TestMethod() string {return "I am great"}`

	method := Method{Code: code, Docs: docs}
	lines := method.Lines()

	require.Equal(t, "// TestMethod is great", lines[0])
	require.Equal(t, "func TestMethod() string {return \"I am great\"}", lines[1])
}

func TestParseDeclaredTypes(t *testing.T) {
	declaredTypes := ParseDeclaredTypes(src)

	require.Equal(t, declaredType{
		Name:    "Person",
		Package: "main",
	},
		declaredTypes[0])
	require.Equal(t, declaredType{
		Name:    "SomeType",
		Package: "main",
	},
		declaredTypes[1])
}

// Ensure that type declarations grouped in a single block are all discovered.
func TestParseDeclaredTypes_MultiSpec(t *testing.T) {
	multiSrc := []byte(`package main
type (
    First int
    Second string
)`)
	types := ParseDeclaredTypes(multiSrc)
	require.Equal(t, []declaredType{
		{Name: "First", Package: "main"},
		{Name: "Second", Package: "main"},
	}, types)
}

func TestParseEmbeddingGraph(t *testing.T) {
	callGraph := ParseEmbeddingGraph(src)
	require.Equal(t, "Person", callGraph["Turing"][0])
}

// Verify embedded structs referenced from other packages are captured.
func TestParseEmbeddingGraph_SelectorExpr(t *testing.T) {
	extSrc := []byte(`package main
import "otherpkg"
type MyStruct struct {
    otherpkg.External
    *otherpkg.Pointer
}`)
	graph := ParseEmbeddingGraph(extSrc)
	require.Contains(t, graph, "MyStruct")
	require.ElementsMatch(t, []string{"External", "Pointer"}, graph["MyStruct"])
}

// Verify embedded generic struct instantiations are detected.
func TestParseEmbeddingGraph_Generic(t *testing.T) {
	src := []byte(`package main
type Generic[T any] struct{}
type MyStruct struct {
       Generic[int]
}`)
	graph := ParseEmbeddingGraph(src)
	require.Contains(t, graph, "MyStruct")
	require.ElementsMatch(t, []string{"Generic"}, graph["MyStruct"])
}

// Verify pointer generic embeddings are also handled.
func TestParseEmbeddingGraph_GenericPointer(t *testing.T) {
	src := []byte(`package main
type Generic[T any] struct{}
type MyStruct struct {
       *Generic[int]
}`)
	graph := ParseEmbeddingGraph(src)
	require.Contains(t, graph, "MyStruct")
	require.ElementsMatch(t, []string{"Generic"}, graph["MyStruct"])
}

func TestParseStruct(t *testing.T) {
	methods, imports, typeDoc, _ := ParseStruct(src, "Person", true, true, "", nil, "", false, nil, false)

	require.Equal(t, "Name() (string)", methods[0].Code)

	imp := imports[0]
	trimmedImp := strings.TrimSpace(imp)

	require.Equal(t, `notmain "fmt"`, trimmedImp)
	require.Equal(t, "Person ...", typeDoc)
}

func TestParseStructWithImportModule(t *testing.T) {
	methods, imports, typeDoc, _ := ParseStruct(src, "Person", true, true, "", nil, "github.com/test/test", false, nil, false)

	require.Equal(t, "Name() (string)", methods[0].Code)

	imp, module := imports[0], imports[1]
	trimmedImp := strings.TrimSpace(imp)

	require.Equal(t, `notmain "fmt"`, trimmedImp)
	require.Equal(t, `. "github.com/test/test"`, module)
	require.Equal(t, "Person ...", typeDoc)
}

func TestParseStructWithNotExported(t *testing.T) {
	methods, _, _, _ := ParseStruct(src, "Person", true, true, "", nil, "github.com/test/test", true, nil, false)

	var oneExists, twoExists bool
	for _, method := range methods {
		if method.Code == "unexportedFuncOne() (bool)" {
			oneExists = true
		}

		if method.Code == "unexportedFuncTwo() (string)" {
			twoExists = true
		}
	}

	require.True(t, oneExists)
	require.True(t, twoExists)
}

func TestParseStructWithPromoted(t *testing.T) {
	callGraph := map[string]struct{}{
		"Person": {},
	}
	methods, imports, typeDoc, _ := ParseStruct(src, "Turing", true, true, "", nil, "", false, callGraph, true)
	t.Log(methods)
	t.Log(imports)
	t.Log(typeDoc)

	require.Equal(t, "Name() (string)", methods[0].Code)

	imp := imports[0]
	trimmedImp := strings.TrimSpace(imp)

	require.Equal(t, `notmain "fmt"`, trimmedImp)
	require.Equal(t, "Turing is a person", typeDoc)
}

func TestGetReceiverTypeName(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.Nil(t, err, "ParseFile returned an error")

	hasPersonFuncDecl := false
	for _, d := range a.Decls {
		typeName, fd := GetReceiverTypeName(src, d)
		if typeName == "" {
			continue
		}
		switch typeName {
		case "Person":
			require.NotNil(t, fd, "receiver type with name %s had a nil func decl")
			// OK
			hasPersonFuncDecl = true
		}
	}

	require.True(t, hasPersonFuncDecl, "Never registered a func decl with the `Person` receiver type")
}

func TestFormatFieldList(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.Nil(t, err, "ParseFile returned an error")

	for _, d := range a.Decls {
		if a, fd := GetReceiverTypeName(src, d); a == "Person" {
			methodName := fd.Name.String()
			params := FormatFieldList(src, fd.Type.Params, "main", nil)
			results := FormatFieldList(src, fd.Type.Results, "main", nil)

			var expectedParams []string
			var expectedResults []string
			switch methodName {
			case "Name":
				expectedResults = []string{"string"}
			case "SetName":
				expectedParams = []string{"name string"}
			case "Age":
				expectedResults = []string{"int"}
			case "SetAge":
				expectedParams = []string{"age int"}
			case "AgeAndName":
				expectedResults = []string{"int", "string"}
			case "SetAgeAndName":
				expectedParams = []string{"name string", "age int"}
			case "GetNameAndTelephone":
				expectedResults = []string{"name, telephone string"}
			case "SetNameAndTelephone":
				expectedParams = []string{"name, telephone string"}
			case "ReturnPointer":
				expectedResults = []string{"*notmain.Formatter"}
			case "ReturnNoPointer":
				expectedResults = []string{"notmain.Formatter"}
			case "unexportedFuncOne":
				expectedResults = []string{"bool"}
			case "unexportedFuncTwo":
				expectedResults = []string{"string"}
			case "ArgumentNoPointer":
				expectedParams = []string{"formatter notmain.Formatter"}
			case "ArgumentPointer":
				expectedParams = []string{"formatter *notmain.Formatter"}
			case "SecondArgumentPointer":
				expectedParams = []string{"abc int", "formatter *notmain.Formatter"}
			}
			require.Equal(t, expectedParams, params)
			require.Equal(t, expectedResults, results)
		}
	}
}

func TestNoCopyTypeDocs(t *testing.T) {
	_, _, typeDoc, _ := ParseStruct(src, "Person", true, false, "", nil, "", false, nil, false)
	require.Equal(t, "", typeDoc)
}

func TestMakeInterface(t *testing.T) {
	methods := []string{"// MyMethod does cool stuff", "MyMethod(string) example.Example"}
	imports := []string{`"github.com/example/example"`}
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "MyInterface does cool stuff", "", methods, imports)
	require.Nil(t, err, "MakeInterface returned an error")

	expected := `// DO NOT EDIT: Auto generated

package pkg

import (
	"github.com/example/example"
)

// MyInterface does cool stuff
type MyInterface interface {
	// MyMethod does cool stuff
	MyMethod(string) example.Example
}
`

	require.Equal(t, expected, string(b))
}

func TestMakeWithoutInterfaceComment(t *testing.T) {
	methods := []string{"// MyMethod does cool stuff", "MyMethod(string) example.Example"}
	imports := []string{`"github.com/example/example"`}
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "", "", methods, imports)
	require.Nil(t, err, "MakeInterface returned an error")

	expected := `// DO NOT EDIT: Auto generated

package pkg

import (
	"github.com/example/example"
)

type MyInterface interface {
	// MyMethod does cool stuff
	MyMethod(string) example.Example
}
`

	require.Equal(t, expected, string(b))
}

func TestMakeInterfaceWithGoGenerate(t *testing.T) {
	methods := []string{"// MyMethod does cool stuff", "MyMethod(string) example.Example"}
	imports := []string{`"github.com/example/example"`}
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "go:generate MyInterface does cool stuff", "", methods, imports)
	require.Nil(t, err, "MakeInterface returned an error")

	expected := `// DO NOT EDIT: Auto generated

package pkg

import (
	"github.com/example/example"
)

//go:generate MyInterface does cool stuff
type MyInterface interface {
	// MyMethod does cool stuff
	MyMethod(string) example.Example
}
`

	require.Equal(t, expected, string(b))
}

func TestMakeInterfaceMultiLineIfaceComment(t *testing.T) {
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "MyInterface does cool stuff.\nWith multi-line comments.", "", nil, nil)
	require.Nil(t, err, "MakeInterface returned an error:", err)

	expected := `// DO NOT EDIT: Auto generated

package pkg

// MyInterface does cool stuff.
// With multi-line comments.
type MyInterface interface {
}
`

	require.Equal(t, expected, string(b))
}

func Test_validate_struct_types(t *testing.T) {
	types := []declaredType{}
	tt := []struct {
		name   string
		inpSet func()
		stType string
		exp    bool
	}{
		{
			name: "valid struct type present in file",
			inpSet: func() {
				types = append(types, declaredType{Name: t.Name(), Package: t.Name()})
			},
			stType: t.Name(),
			exp:    true,
		},
		{
			name:   "struct not present",
			inpSet: func() {},
			stType: fmt.Sprintf("%s-1", t.Name()),
		},
		{
			name: "case mismatch",
			inpSet: func() {
				types = append(types, declaredType{Name: "MyStruct", Package: "pkg"})
			},
			stType: "mystruct",
			exp:    false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// populate data in test input
			tc.inpSet()
			// test
			got := validateStructType(types, tc.stType)
			// validate
			require.Equal(t, tc.exp, got)
		})

	}

}

func TestDeclaredTypeFullname(t *testing.T) {
	dt := declaredType{Name: "Test", Package: "pkg"}
	require.Equal(t, "pkg.Test", dt.Fullname())
}

func TestGetTypeDeclarationName_Valid(t *testing.T) {
	src := []byte("package main\ntype MyType int")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	var found string
	for _, d := range file.Decls {
		name := GetTypeDeclarationName(d)
		if name != "" {
			found = name
			break
		}
	}
	require.Equal(t, "MyType", found)
}

func TestGetTypeDeclarationName_NonTypeDecl(t *testing.T) {
	src := []byte("package main\nfunc main() {}")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	for _, d := range file.Decls {
		name := GetTypeDeclarationName(d)
		// For non-type declarations, the function should return an empty string.
		require.Equal(t, "", name)
	}
}

func TestGetReceiverType_NotMethod(t *testing.T) {
	// Create a FuncDecl with no receiver (i.e. a plain function)
	fd := &ast.FuncDecl{
		Recv: nil,
		Name: ast.NewIdent("Func"),
	}
	_, err := GetReceiverType(fd)
	require.Error(t, err)
}

func TestFormatCodeValid(t *testing.T) {
	code := "package main\nfunc main(){println(\"hello\")}"
	formatted, err := FormatCode(code)
	require.NoError(t, err)
	// Check that the formatted code contains the package declaration.
	require.Contains(t, string(formatted), "package main")
}

func TestFormatCodeInvalid(t *testing.T) {
	// Providing a code fragment that is not valid Go code.
	code := "not a valid go code"
	_, err := FormatCode(code)
	require.Error(t, err)
}

func TestFormatFieldList_Nil(t *testing.T) {
	parts := FormatFieldList([]byte(""), nil, "main", nil)
	require.Nil(t, parts)
}

func TestMakeStructNotFound(t *testing.T) {
	// Create a temporary file with source that does not declare the expected struct.
	tmpFile, err := os.CreateTemp("", "test*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	content := []byte("package main\nfunc Foo() {}")
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	_, err = Make(MakeOptions{
		Files:      []string{tmpFile.Name()},
		StructType: "NonExistent",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "TestIface",
	})
	require.Error(t, err)
	// Update expected substring to include the quotes
	require.Contains(t, err.Error(), `"NonExistent" structtype not found`)
}

func TestMakeFileNotFound(t *testing.T) {
	// Provide a filename that does not exist.
	_, err := Make(MakeOptions{
		Files:      []string{"non_existing_file.go"},
		StructType: "Foo",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "TestIface",
	})
	require.Error(t, err)
}

// TestParseDeclaredTypesEmpty ensures that a source with no type declarations returns an empty slice.
func TestParseDeclaredTypesEmpty(t *testing.T) {
	src := []byte("package main\nfunc Foo() {}")
	types := ParseDeclaredTypes(src)
	require.Empty(t, types)
}

// TestFormatFieldList_MultipleNames verifies that parameters with multiple names are formatted correctly.
func TestFormatFieldList_MultipleNames(t *testing.T) {
	src := []byte(`package main
func Foo(a, b int) int { return 0 }`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "Foo" {
			fd = f
			break
		}
	}
	require.NotNil(t, fd)
	params := FormatFieldList(src, fd.Type.Params, "main", nil)
	// Expect parameters to be formatted as "a, b int"
	require.Contains(t, params, "a, b int")
}

// TestGetReceiverTypeName_NonPointer checks that a non-pointer receiver is handled without stripping extra characters.
func TestGetReceiverTypeName_NonPointer(t *testing.T) {
	src := []byte(`package main
type MyStruct struct {}
func (m MyStruct) Foo() {}`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	found := false
	for _, d := range file.Decls {
		typeName, fd := GetReceiverTypeName(src, d)
		if typeName == "MyStruct" {
			found = true
			// Should not have a leading '*' since receiver is not a pointer.
			require.Equal(t, "MyStruct", typeName)
			require.Equal(t, "Foo", fd.Name.Name)
		}
	}
	require.True(t, found, "Expected to find a receiver with type 'MyStruct'")
}

// TestMakeExcludeMethod ensures that methods listed in the exclusion set are omitted.
func TestMakeExcludeMethod(t *testing.T) {
	// Create a temporary file with a struct that has two methods: Foo and Bar.
	tmpFile, err := os.CreateTemp("", "test_exclude_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	src := []byte(`package main
type MyStruct struct {}
func (m *MyStruct) Foo() {}
func (m *MyStruct) Bar() {}
`)
	_, err = tmpFile.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	result, err := Make(MakeOptions{
		Files:          []string{tmpFile.Name()},
		StructType:     "MyStruct",
		Comment:        "Test Comment",
		PkgName:        "main",
		IfaceName:      "MyIface",
		ExcludeMethods: []string{"Bar"},
		CopyDocs:       true,
	})
	require.NoError(t, err)
	outStr := string(result)
	require.Contains(t, outStr, "Foo()")
	require.NotContains(t, outStr, "Bar()")
}

// TestMakeDuplicateMethods verifies that if the same method is present in multiple files, it appears only once.
func TestMakeDuplicateMethods(t *testing.T) {
	src1 := []byte(`package main
type MyStruct struct {}
func (m *MyStruct) Foo() {}
`)
	src2 := []byte(`package other
type MyStruct struct {}
func (m *MyStruct) Foo() {}
`)
	tmpFile1, err := os.CreateTemp("", "dup1_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile1.Name()) }()
	_, err = tmpFile1.Write(src1)
	require.NoError(t, err)
	require.NoError(t, tmpFile1.Close())

	tmpFile2, err := os.CreateTemp("", "dup2_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile2.Name()) }()
	_, err = tmpFile2.Write(src2)
	require.NoError(t, err)
	require.NoError(t, tmpFile2.Close())

	result, err := Make(MakeOptions{
		Files:      []string{tmpFile1.Name(), tmpFile2.Name()},
		StructType: "MyStruct",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "MyIface",
		CopyDocs:   false,
	})
	require.NoError(t, err)
	outStr := string(result)
	// The method Foo() should appear only once.
	count := strings.Count(outStr, "Foo()")
	require.Equal(t, 1, count)
}

// TestMake_NoMethods creates a temporary Go file that defines a struct with no methods.
// This ensures that Make() handles the case when there are no methods to include.
func TestMake_NoMethods(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_nomethods_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// The source contains a struct "MyStruct" with no methods.
	content := []byte(`package main
type MyStruct struct {}`)
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	result, err := Make(MakeOptions{
		Files:      []string{tmpFile.Name()},
		StructType: "MyStruct",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "MyIface",
	})
	require.NoError(t, err)
	outStr := string(result)
	// Verify the interface declaration exists but contains no methods.
	require.Contains(t, outStr, "type MyIface interface {")
	// Extract the block inside the interface declaration.
	lines := strings.Split(outStr, "\n")
	inIface := false
	methodCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "type") && strings.Contains(trimmed, "interface") {
			inIface = true
			continue
		}
		if inIface {
			if trimmed == "}" {
				break
			}
			if trimmed != "" {
				methodCount++
			}
		}
	}
	require.Equal(t, 0, methodCount)
}

// TestFormatFieldList_Replaced_WithNames verifies that FormatFieldList
// replaces a field type that matches a declared type (with names present)
func TestFormatFieldList_Replaced_WithNames(t *testing.T) {
	// The source has a parameter "x []MyType"
	src := []byte(`package main
func Foo(x []MyType) {}`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	// Find the function declaration for Foo
	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "Foo" {
			fd = f
			break
		}
	}
	require.NotNil(t, fd)

	// declaredTypes contains MyType but with a different package so that replacement occurs.
	declaredTypes := []declaredType{
		{Name: "MyType", Package: "other"},
	}
	// Call FormatFieldList with pkgName "main"
	params := FormatFieldList(src, fd.Type.Params, "main", declaredTypes)
	// Expect the type "[]MyType" to be replaced with "[]other.MyType"
	require.Len(t, params, 1)
	require.Equal(t, "x []other.MyType", params[0])
}

// TestFormatFieldList_UnnamedReturn verifies FormatFieldList on a return field list with no names.
func TestFormatFieldList_UnnamedReturn(t *testing.T) {
	// The function returns an unnamed []MyType.
	src := []byte(`package main
func Foo() []MyType { return nil }`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	// Find the function declaration for Foo
	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "Foo" {
			fd = f
			break
		}
	}
	require.NotNil(t, fd)

	declaredTypes := []declaredType{
		{Name: "MyType", Package: "other"},
	}
	results := FormatFieldList(src, fd.Type.Results, "main", declaredTypes)
	// When there are no names, the type string itself is added.
	require.Len(t, results, 1)
	require.Equal(t, "[]other.MyType", results[0])
}

// TestFormatFieldList_NoModifier covers the branch in FormatFieldList()
// where there is no regex match and t is set via dt.Fullname().
func TestFormatFieldList_NoModifier(t *testing.T) {
	// Source with a parameter type "MyType" (no modifier like "*" or "[]")
	src := []byte(`package main
func Foo(x MyType) {}`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "Foo" {
			fd = f
			break
		}
	}
	require.NotNil(t, fd)

	// declaredTypes includes MyType with a package different than "main"
	declaredTypes := []declaredType{
		{Name: "MyType", Package: "other"},
	}
	// When pkgName ("main") is different from dt.Package ("other"),
	// the type should be replaced with dt.Fullname() ("other.MyType").
	params := FormatFieldList(src, fd.Type.Params, "main", declaredTypes)
	require.Len(t, params, 1)
	require.Equal(t, "x other.MyType", params[0])
}

// TestFormatFieldList_StripDestPkg verifies that package prefixes matching
// the destination package are removed even when preceding characters are
// brackets or other non-word tokens.
func TestFormatFieldList_StripDestPkg(t *testing.T) {
	src := []byte(`package foo
import "bar"
type Foo struct{}
func (f *Foo) Use(x []bar.Type) {}`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "Use" {
			fd = f
			break
		}
	}
	require.NotNil(t, fd)

	params := FormatFieldList(src, fd.Type.Params, "bar", nil)
	require.Len(t, params, 1)
	require.Equal(t, "x []Type", params[0])
}

// TestMake_StructTypeNotFound_EmptyFile covers the branch in Make()
// where the declared struct type is not found in the input files.
func TestMake_StructTypeNotFound_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_empty_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// File with no type declarations.
	content := []byte("package main\n")
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	_, err = Make(MakeOptions{
		Files:      []string{tmpFile.Name()},
		StructType: "NonExistent",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "TestIface",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `"NonExistent" structtype not found in input files`)
}

// TestMake_DuplicateImports verifies that when multiple files import the same package,
// the for-loop over imports in Make() deduplicates them.
func TestMake_DuplicateImports(t *testing.T) {
	// Create first temporary file with an import and a method that uses fmt.Stringer.
	src1 := []byte(`package main
import "fmt"
type MyStruct struct {}
func (m *MyStruct) Foo() fmt.Stringer { return nil }`)
	tmpFile1, err := os.CreateTemp("", "dupimports1_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile1.Name()) }()
	_, err = tmpFile1.Write(src1)
	require.NoError(t, err)
	require.NoError(t, tmpFile1.Close())

	// Create second temporary file with the same import and a method that uses fmt.Stringer.
	src2 := []byte(`package main
import "fmt"
type MyStruct struct {}
func (m *MyStruct) Bar() fmt.Stringer { return nil }`)
	tmpFile2, err := os.CreateTemp("", "dupimports2_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile2.Name()) }()
	_, err = tmpFile2.Write(src2)
	require.NoError(t, err)
	require.NoError(t, tmpFile2.Close())

	result, err := Make(MakeOptions{
		Files:      []string{tmpFile1.Name(), tmpFile2.Name()},
		StructType: "MyStruct",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "TestIface",
		CopyDocs:   false,
	})
	require.NoError(t, err)

	outStr := string(result)
	// Check that the "fmt" import appears only once.
	count := strings.Count(outStr, `"fmt"`)
	require.Equal(t, 1, count)
}

func TestMake_WithPromoted(t *testing.T) {
	// Create a temporary file with a struct that embeds another struct.
	tmpFile, err := os.CreateTemp("", "test_withpromoted_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	src := []byte(`package main
type EmbeddedStruct struct {}
func (e *EmbeddedStruct) EmbeddedMethod() {}

type MyStruct struct {
	EmbeddedStruct
}
`)
	_, err = tmpFile.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	result, err := Make(MakeOptions{
		Files:        []string{tmpFile.Name()},
		StructType:   "MyStruct",
		Comment:      "Test Comment",
		PkgName:      "main",
		IfaceName:    "MyIface",
		WithPromoted: true,
	})
	require.NoError(t, err)
	outStr := string(result)

	// Check that the promoted method from EmbeddedStruct is included.
	require.Contains(t, outStr, "EmbeddedMethod()")
}

func TestMake_WithPromotedPointer(t *testing.T) {
	// Create a temporary file with a struct that embeds a pointer to another struct.
	tmpFile, err := os.CreateTemp("", "test_withpromoted_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	src := []byte(`package main
type EmbeddedStruct struct {}
func (e *EmbeddedStruct) EmbeddedMethod() {}

type MyStruct struct {
	*EmbeddedStruct
}
`)
	_, err = tmpFile.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	result, err := Make(MakeOptions{
		Files:        []string{tmpFile.Name()},
		StructType:   "MyStruct",
		Comment:      "Test Comment",
		PkgName:      "main",
		IfaceName:    "MyIface",
		WithPromoted: true,
	})
	require.NoError(t, err)
	outStr := string(result)

	// Check that the promoted method from EmbeddedStruct is included.
	require.Contains(t, outStr, "EmbeddedMethod()")
}

func TestMake_WithPromotedOverride(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_withpromoted_override_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	src := []byte(`package main
type Base struct{}
func (b *Base) Foo() string { return "" }

type Sub struct {
        Base
}
func (s *Sub) Foo() int { return 0 }
`)
	_, err = tmpFile.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	result, err := Make(MakeOptions{
		Files:        []string{tmpFile.Name()},
		StructType:   "Sub",
		Comment:      "Test Comment",
		PkgName:      "main",
		IfaceName:    "MyIface",
		WithPromoted: true,
	})
	require.NoError(t, err)
	outStr := string(result)

	require.Contains(t, outStr, "Foo() int")
	require.NotContains(t, outStr, "Foo() string")
}

// TestParseDeclaredTypes_Fatal runs ParseDeclaredTypes with invalid Go code.
// Because ParseDeclaredTypes calls log.Fatal on parse errors, we run this in a subprocess.
func TestParseDeclaredTypes_Fatal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		// Provide invalid Go source to force a parse error
		ParseDeclaredTypes([]byte("invalid go code"))
		// This point should not be reached because log.Fatal should exit.
		return
	}
	// Re-run this test in a subprocess so that the os.Exit call can be observed.
	cmd := exec.Command(os.Args[0], "-test.run=TestParseDeclaredTypes_Fatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	// An exit error is expected due to log.Fatal.
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return // Test passed.
	}
	t.Fatalf("ParseDeclaredTypes did not exit as expected")
}

// TestParseStruct_Fatal runs ParseStruct with invalid Go source.
// Since ParseStruct calls log.Fatal on parse errors, we capture that via a subprocess.
func TestParseStruct_Fatal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		// Provide invalid source code to trigger parser.ParseFile error inside ParseStruct.
		ParseStruct([]byte("invalid go code"), "Foo", true, true, "", nil, "", false, nil, false)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestParseStruct_Fatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return // Test passed.
	}
	t.Fatalf("ParseStruct did not exit as expected")
}

func TestGenericsSupport(t *testing.T) {
	src := []byte(`package generic

type Box[T any] struct{}

func (b *Box[T]) Add(v T) {}
func (b *Box[T]) Get() T { var zero T; return zero }
`)

	methods, _, _, typeParams := ParseStruct(src, "Box", true, true, "generic", nil, "", false, nil, false)
	require.Equal(t, "[T any]", typeParams)
	require.Equal(t, "Add(v T)", methods[0].Code)
	require.Equal(t, "Get() (T)", methods[1].Code)

	tmpFile, err := os.CreateTemp("", "generic_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_, err = tmpFile.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	result, err := Make(MakeOptions{
		Files:      []string{tmpFile.Name()},
		StructType: "Box",
		Comment:    "Test Comment",
		PkgName:    "generic",
		IfaceName:  "BoxIface",
		CopyDocs:   false,
	})
	require.NoError(t, err)
	expected := `// Test Comment

package generic

type BoxIface[T any] interface {
	Add(v T)
	Get() T
}
`
	require.Equal(t, expected, string(result))
}

func TestGenericCrossPackageParam(t *testing.T) {
	src := []byte(`package foo

type Generic[T any] struct{}
type Example[T any] struct{}

func (e *Example[T]) Use(g Generic[T]) {}
`)

	types := ParseDeclaredTypes(src)
	methods, _, _, _ := ParseStruct(src, "Example", true, true, "bar", types, "", false, nil, false)

	require.Equal(t, "Use(g foo.Generic[T])", methods[0].Code)
}

func TestGetTypeDeclarationName_NonTypeSpec(t *testing.T) {
	gd := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.ValueSpec{}}}
	name := GetTypeDeclarationName(gd)
	require.Equal(t, "", name)
}

func TestParseEmbeddingGraph_NonStruct(t *testing.T) {
	src := []byte("package main\ntype Alias int")
	graph := ParseEmbeddingGraph(src)
	require.Empty(t, graph)
}

func TestParseEmbeddingGraph_ParseError(t *testing.T) {
	if os.Getenv("BE_PEG_CRASHER") == "1" {
		ParseEmbeddingGraph([]byte("invalid"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestParseEmbeddingGraph_ParseError")
	cmd.Env = append(os.Environ(), "BE_PEG_CRASHER=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return
	}
	t.Fatalf("ParseEmbeddingGraph did not exit as expected")
}

func TestMake_DuplicateEmbeddedStructs(t *testing.T) {
	src := []byte(`package main
    type B struct{}
    type A struct{B; B}`)
	tmp, err := os.CreateTemp("", "dupembed_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	_, err = Make(MakeOptions{
		Files:        []string{tmp.Name()},
		StructType:   "A",
		Comment:      "c",
		PkgName:      "main",
		IfaceName:    "I",
		WithPromoted: true,
	})
	require.NoError(t, err)
}

func TestMake_WithCopyTypeDoc(t *testing.T) {
	src := []byte(`package docpkg
    // MyStruct does things
    type MyStruct struct{}`)
	tmp, err := os.CreateTemp("", "copydoc_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	result, err := Make(MakeOptions{
		Files:        []string{tmp.Name()},
		StructType:   "MyStruct",
		Comment:      "header",
		PkgName:      "docpkg",
		IfaceName:    "MyIface",
		CopyTypeDoc:  true,
		IfaceComment: "iface comment",
	})
	require.NoError(t, err)
	require.Contains(t, string(result), "// iface comment\n// MyStruct does things")
}

func TestMakeInterfaceError(t *testing.T) {
	src := []byte(`package pkg
type S struct{}`)
	tmp, err := os.CreateTemp("", "ifaceerr_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())
	_, err = Make(MakeOptions{Files: []string{tmp.Name()}, StructType: "S", Comment: "c", PkgName: "pkg", IfaceName: "1bad"})
	require.Error(t, err)
}

func TestParseStruct_DuplicateMethod(t *testing.T) {
	src := []byte(`package main
type Foo struct{}
func (f *Foo) Bar() {}
func (f *Foo) Bar() {}`)
	methods, _, _, _ := ParseStruct(src, "Foo", true, true, "main", nil, "", false, nil, false)
	count := 0
	for _, m := range methods {
		if strings.HasPrefix(m.Code, "Bar(") {
			count++
		}
	}
	require.Equal(t, 1, count)
}
