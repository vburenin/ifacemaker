package maker

import (
	"strings"
	"testing"
)

func check(value, pattern string, t *testing.T) {
	if value != pattern {
		t.Fatalf("Value %s did not match expected pattern %s", value, pattern)
	}
}

func TestLines(t *testing.T) {
	docs := []string{`// TestMethod is great`}
	code := `func TestMethod() string {return "I am great"}`
	method := Method{Code: code, Docs: docs}
	lines := method.Lines()
	check(lines[0], "// TestMethod is great", t)
	check(lines[1], "func TestMethod() string {return \"I am great\"}", t)
}

func TestParseStruct(t *testing.T) {
	src := []byte(`package main
	    
	    import (
		"fmt"
	    )

	    // Person ...
	    type Person struct {
		name string
	    }

	    // Name ...
	    func (p *Person) Name() string {
		return p.name
	    }`)
	methods, imports := ParseStruct(src, "Person", true)
	check(methods[0].Code, "Name() string", t)
	imp := imports[0]
	trimmedImp := strings.TrimSpace(imp)
	expected := "\"fmt\""
	check(trimmedImp, expected, t)
}

func Test
