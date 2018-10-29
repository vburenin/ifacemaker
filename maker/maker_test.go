package maker

import (
	"go/parser"
	"go/token"
	"log"
	"strings"
	"testing"
)

var (
	src = []byte(`
		package main

		import (
		    "fmt"
		)

		// Person ...
		type Person struct {
		    name string
		    age int
		    telephone string
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
		func (p *Person) SetAge(age int) int {
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

		func SomeFunction() string {
		    return "Something"
		}`)
)

func mustBeEqual(value, pattern string, t *testing.T) {
	if value != pattern {
		t.Fatalf("Value %s did not match expected pattern %s", value, pattern)
	}
}

func TestLines(t *testing.T) {
	docs := []string{`// TestMethod is great`}
	code := `func TestMethod() string {return "I am great"}`
	method := Method{Code: code, Docs: docs}
	lines := method.Lines()
	mustBeEqual(lines[0], "// TestMethod is great", t)
	mustBeEqual(lines[1], "func TestMethod() string {return \"I am great\"}", t)
}

func TestParseStruct(t *testing.T) {
	methods, imports := ParseStruct(src, "Person", true)
	mustBeEqual(methods[0].Code, "Name() (string)", t)
	imp := imports[0]
	trimmedImp := strings.TrimSpace(imp)
	expected := "\"fmt\""
	mustBeEqual(trimmedImp, expected, t)
}

func TestGetReceiverTypeName(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err.Error())
	}
	hasPersonFuncDecl := false
	for _, d := range a.Decls {
		typeName, fd := GetReceiverTypeName(src, d)
		if typeName == "" {
			continue
		}
		switch typeName {
		case "Person":
			if fd == nil {
				t.Fatalf("receiver type with name %s had a nil func decl", typeName)
			}
			// OK
			hasPersonFuncDecl = true
		}
	}
	if !hasPersonFuncDecl {
		t.Fatalf("Never registered a func decl with the `Person` receiver type")
	}
}

func TestIsFunctionPrivate(t *testing.T) {
	functionNames := map[string]bool{"_someFunc": true, "herp": true, "_SomeFunc": true, "SomeFunc": false, "ZooFunc": false}
	for name, expected := range functionNames {
		if isFunctionPrivate(name) != expected {
			t.Fatalf("Function name %s != expected value %+v", name, expected)
		}
	}

}

func TestFormatFieldList(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, d := range a.Decls {
		if a, fd := GetReceiverTypeName(src, d); a == "Person" {
			methodName := fd.Name.String()
			params := FormatFieldList(src, fd.Type.Params)
			results := FormatFieldList(src, fd.Type.Results)
			switch methodName {
			case "Name":
				expectedParam := []string{}
				expectedResults := []string{"string"}
				if !compareStrArrays(expectedParam, params, t) || !compareStrArrays(expectedResults, results, t) {
					t.Fatalf("Name did not have the expected params and/or results")
				}
			case "Age":
				expectedParams := []string{}
				expectedResults := []string{"int"}
				if !compareStrArrays(expectedParams, params, t) || !compareStrArrays(expectedResults, results, t) {
					t.Fatalf("Age did not have the expected params and/or results")
				}
			case "SetName":
				expectedParams := []string{"name string"}
				expectedResults := []string{}
				if !compareStrArrays(expectedParams, params, t) || !compareStrArrays(expectedResults, results, t) {
					t.Fatalf("SetName did not have the expected params and/or results")
				}
			case "SetAgeAndName":
				expectedParams := []string{"name string", "age int"}
				expectedResults := []string{}
				if !compareStrArrays(expectedParams, params, t) || !compareStrArrays(expectedResults, results, t) {
					t.Fatalf("SetAgeAndName did not have the expected params and/or results")
				}
			case "GetNameAndTelephone":
				expectedParams := []string{}
				expectedResults := []string{"name, telephone string"}
				if !compareStrArrays(expectedParams, params, t) || !compareStrArrays(expectedResults, results, t) {
					t.Fatalf("GetNameAndTelephone did not have the expected params and/or results")
				}
			case "SetNameAndTelephone":
				expectedParams := []string{"name, telephone string"}
				expectedResults := []string{}
				if !compareStrArrays(expectedParams, params, t) || !compareStrArrays(expectedResults, results, t) {
					t.Fatalf("SetNameAndTelephone did not have the expected params and/or results")
				}
			}
		}
	}
}

func compareStrArrays(actual []string, expected []string, t *testing.T) bool {
	if len(actual) != len(expected) {
		t.Logf("compareStrArrays received two different lengths of fields, expected:|%+v| was not equal to actual |%+v|. actual length:%d, expected length:%d", expected, actual, len(actual), len(expected))
		return false
	}
	for i := 0; i < len(actual); i++ {
		if actual[i] != expected[i] {
			t.Logf("compareStrArrays expected:|%+v| was not equal to actual |%+v|", expected, actual)
			return false
		}
	}
	return true
}
