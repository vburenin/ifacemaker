package maker

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	src = []byte(`
	package main

	import (
		notmain "fmt"
	)
	
	// Person ...
	type Person struct {
		name string
		age int
		telephone string
		noPointer notmain.Formatter
		pointer *notmain.Formatter
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
		return p.Age
	}
	
	// Age ...
	func (p *Person) SetAge(age int) {
		p.Age = age
	}
	
	// AgeAndName ...
	func (p *Person) AgeAndName() (int, string) {
		return p.age, p.name
	}
	
	func (p *Person) SetAgeAndName(name string, age int) {
		p.name = name
		p.age = age
	}
	
	// TelephoneAndName ...
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
	
	func (p *Person) ArgumentNoPointer(formatter notmain.Formatter)  {
	
	}
	
	func (p *Person) ArgumentPointer(formatter *notmain.Formatter)  {
	
	}
	
	func (p *Person) SecondArgumentPointer(abc int, formatter *notmain.Formatter)  {
	
	}
	
	func SomeFunction() string {
		return "Something"
	}

	type SomeType struct {}`)

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
	methods, imports, typeDoc := ParseStruct(src, "Person", true, true, "", nil)

	assert.Equal(t, "Name() (string)", methods[0].Code)

	imp := imports[0]
	trimmedImp := strings.TrimSpace(imp)

	assert.Equal(t, `notmain "fmt"`, trimmedImp)
	assert.Equal(t, "Person ...", typeDoc)
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
			case "ArgumentNoPointer":
				expectedParams = []string{"formatter notmain.Formatter"}
			case "ArgumentPointer":
				expectedParams = []string{"formatter *notmain.Formatter"}
			case "SecondArgumentPointer":
				expectedParams = []string{"abc int", "formatter *notmain.Formatter"}
			}
			assert.Equalf(t, expectedParams, params, "%s must have the expected params", methodName)
			assert.Equalf(t, expectedResults, results, "%s must have the expected results", methodName)
		}
	}
}

func TestNoCopyTypeDocs(t *testing.T) {
	_, _, typeDoc := ParseStruct(src, "Person", true, false, "", nil)
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
