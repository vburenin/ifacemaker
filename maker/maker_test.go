package maker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type SomeType struct{}`)
)

func TestLines(t *testing.T) {
	docs := []string{`// TestMethod is great`}
	code := `func TestMethod() string {return "I am great"}`

	method := Method{Code: code, Docs: docs}
	lines := method.Lines()

	assert.Equal(t, "// TestMethod is great", lines[0])
	assert.Equal(t, "func TestMethod() string {return \"I am great\"}", lines[1])
}

func TestParseDeclaredTypes(t *testing.T) {
	declaredTypes := ParseDeclaredTypes(src)

	assert.Equal(t, declaredType{
		Name:    "Person",
		Package: "main",
	},
		declaredTypes[0])
	assert.Equal(t, declaredType{
		Name:    "SomeType",
		Package: "main",
	},
		declaredTypes[1])
}

func TestParseStruct(t *testing.T) {
	methods, imports, typeDoc := ParseStruct(src, "Person", true, true, "", nil, "", false)

	assert.Equal(t, "Name() (string)", methods[0].Code)

	imp := imports[0]
	trimmedImp := strings.TrimSpace(imp)

	assert.Equal(t, `notmain "fmt"`, trimmedImp)
	assert.Equal(t, "Person ...", typeDoc)
}

func TestParseStructWithImportModule(t *testing.T) {
	methods, imports, typeDoc := ParseStruct(src, "Person", true, true, "", nil, "github.com/test/test", false)

	assert.Equal(t, "Name() (string)", methods[0].Code)

	imp, module := imports[0], imports[1]
	trimmedImp := strings.TrimSpace(imp)

	assert.Equal(t, `notmain "fmt"`, trimmedImp)
	assert.Equal(t, `. "github.com/test/test"`, module)
	assert.Equal(t, "Person ...", typeDoc)
}

func TestParseStructWithNotExported(t *testing.T) {
	methods, _, _ := ParseStruct(src, "Person", true, true, "", nil, "github.com/test/test", true)

	var oneExists, twoExists bool
	for _, method := range methods {
		if method.Code == "unexportedFuncOne() (bool)" {
			oneExists = true
		}

		if method.Code == "unexportedFuncTwo() (string)" {
			twoExists = true
		}
	}

	assert.True(t, oneExists)
	assert.True(t, twoExists)
}

func TestGetReceiverTypeName(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	assert.Nil(t, err, "ParseFile returned an error")

	hasPersonFuncDecl := false
	for _, d := range a.Decls {
		typeName, fd := GetReceiverTypeName(src, d)
		if typeName == "" {
			continue
		}
		switch typeName {
		case "Person":
			assert.NotNil(t, fd, "receiver type with name %s had a nil func decl")
			// OK
			hasPersonFuncDecl = true
		}
	}

	assert.True(t, hasPersonFuncDecl, "Never registered a func decl with the `Person` receiver type")
}

func TestFormatFieldList(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	assert.Nil(t, err, "ParseFile returned an error")

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
			assert.Equal(t, expectedParams, params)
			assert.Equal(t, expectedResults, results)
		}
	}
}

func TestNoCopyTypeDocs(t *testing.T) {
	_, _, typeDoc := ParseStruct(src, "Person", true, false, "", nil, "", false)
	assert.Equal(t, "", typeDoc)
}

func TestMakeInterface(t *testing.T) {
	methods := []string{"// MyMethod does cool stuff", "MyMethod(string) example.Example"}
	imports := []string{`"github.com/example/example"`}
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "MyInterface does cool stuff", methods, imports)
	assert.Nil(t, err, "MakeInterface returned an error")

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

	assert.Equal(t, expected, string(b))
}

func TestMakeWithoutInterfaceComment(t *testing.T) {
	methods := []string{"// MyMethod does cool stuff", "MyMethod(string) example.Example"}
	imports := []string{`"github.com/example/example"`}
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "", methods, imports)
	assert.Nil(t, err, "MakeInterface returned an error")

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

	assert.Equal(t, expected, string(b))
}

func TestMakeInterfaceWithGoGenerate(t *testing.T) {
	methods := []string{"// MyMethod does cool stuff", "MyMethod(string) example.Example"}
	imports := []string{`"github.com/example/example"`}
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "go:generate MyInterface does cool stuff", methods, imports)
	assert.Nil(t, err, "MakeInterface returned an error")

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

	assert.Equal(t, expected, string(b))
}

func TestMakeInterfaceMultiLineIfaceComment(t *testing.T) {
	b, err := MakeInterface("DO NOT EDIT: Auto generated", "pkg", "MyInterface", "MyInterface does cool stuff.\nWith multi-line comments.", nil, nil)
	assert.Nil(t, err, "MakeInterface returned an error:", err)

	expected := `// DO NOT EDIT: Auto generated

package pkg

// MyInterface does cool stuff.
// With multi-line comments.
type MyInterface interface {
}
`

	assert.Equal(t, expected, string(b))
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// populate data in test input
			tc.inpSet()
			// test
			got := validateStructType(types, tc.stType)
			// validate
			assert.Equal(t, tc.exp, got)
		})

	}

}

func TestDeclaredTypeFullname(t *testing.T) {
	dt := declaredType{Name: "Test", Package: "pkg"}
	assert.Equal(t, "pkg.Test", dt.Fullname())
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
	assert.Equal(t, "MyType", found)
}

func TestGetTypeDeclarationName_NonTypeDecl(t *testing.T) {
	src := []byte("package main\nfunc main() {}")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)

	for _, d := range file.Decls {
		name := GetTypeDeclarationName(d)
		// For non-type declarations, the function should return an empty string.
		assert.Equal(t, "", name)
	}
}

func TestGetReceiverType_NotMethod(t *testing.T) {
	// Create a FuncDecl with no receiver (i.e. a plain function)
	fd := &ast.FuncDecl{
		Recv: nil,
		Name: ast.NewIdent("Func"),
	}
	_, err := GetReceiverType(fd)
	assert.Error(t, err)
}

func TestFormatCodeValid(t *testing.T) {
	code := "package main\nfunc main(){println(\"hello\")}"
	formatted, err := FormatCode(code)
	assert.NoError(t, err)
	// Check that the formatted code contains the package declaration.
	assert.Contains(t, string(formatted), "package main")
}

func TestFormatCodeInvalid(t *testing.T) {
	// Providing a code fragment that is not valid Go code.
	code := "not a valid go code"
	_, err := FormatCode(code)
	assert.Error(t, err)
}

func TestFormatFieldList_Nil(t *testing.T) {
	parts := FormatFieldList([]byte(""), nil, "main", nil)
	assert.Nil(t, parts)
}

func TestMakeStructNotFound(t *testing.T) {
	// Create a temporary file with source that does not declare the expected struct.
	tmpFile, err := os.CreateTemp("", "test*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("package main\nfunc Foo() {}")
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = Make(MakeOptions{
		Files:      []string{tmpFile.Name()},
		StructType: "NonExistent",
		Comment:    "Test Comment",
		PkgName:    "main",
		IfaceName:  "TestIface",
	})
	assert.Error(t, err)
	// Update expected substring to include the quotes
	assert.Contains(t, err.Error(), `"NonExistent" structtype not found`)
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
	assert.Error(t, err)
}

// TestParseDeclaredTypesEmpty ensures that a source with no type declarations returns an empty slice.
func TestParseDeclaredTypesEmpty(t *testing.T) {
	src := []byte("package main\nfunc Foo() {}")
	types := ParseDeclaredTypes(src)
	assert.Empty(t, types)
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
	assert.Contains(t, params, "a, b int")
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
			assert.Equal(t, "MyStruct", typeName)
			assert.Equal(t, "Foo", fd.Name.Name)
		}
	}
	assert.True(t, found, "Expected to find a receiver with type 'MyStruct'")
}

// TestMakeExcludeMethod ensures that methods listed in the exclusion set are omitted.
func TestMakeExcludeMethod(t *testing.T) {
	// Create a temporary file with a struct that has two methods: Foo and Bar.
	tmpFile, err := os.CreateTemp("", "test_exclude_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	src := []byte(`package main
type MyStruct struct {}
func (m *MyStruct) Foo() {}
func (m *MyStruct) Bar() {}
`)
	_, err = tmpFile.Write(src)
	require.NoError(t, err)
	tmpFile.Close()

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
	assert.Contains(t, outStr, "Foo()")
	assert.NotContains(t, outStr, "Bar()")
}

// TestMakeDuplicateMethods verifies that if the same method is present in multiple files, it appears only once.
func TestMakeDuplicateMethods(t *testing.T) {
	src1 := []byte(`package main
type MyStruct struct {}
func (m *MyStruct) Foo() {}
`)
	src2 := []byte(`package main
type MyStruct struct {}
func (m *MyStruct) Foo() {}
`)
	tmpFile1, err := os.CreateTemp("", "dup1_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile1.Name())
	_, err = tmpFile1.Write(src1)
	require.NoError(t, err)
	tmpFile1.Close()

	tmpFile2, err := os.CreateTemp("", "dup2_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile2.Name())
	_, err = tmpFile2.Write(src2)
	require.NoError(t, err)
	tmpFile2.Close()

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
	assert.Equal(t, 1, count)
}
