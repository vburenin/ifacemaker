package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

var src = `package main

import (
	"fmt"
)

// Person contains data related to a person.
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
}`

var srcFile = os.TempDir() + "/ifacemaker_src.go"

func TestMain(m *testing.M) {
	writeTestSourceFile()

	os.Exit(m.Run())
}

func writeTestSourceFile() {
	f, err := os.OpenFile(srcFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic("Failed to open test source file.")
	}
	defer f.Close()
	_, err = f.WriteString(src)
	if err != nil {
		panic("Failed to write to test source file.")
	}
}

func TestMainAllArgs(t *testing.T) {
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-y", "PersonIface is an interface for Person.", "-D"}
	out := captureStdout(func() {
		main()
	})

	expected := `// DO NOT EDIT: Auto generated

package gen

// PersonIface is an interface for Person.
// Person contains data related to a person.
type PersonIface interface {
	// Name ...
	Name() string
	// SetName ...
	SetName(name string)
	// Age ...
	Age() int
	// Age ...
	SetAge(age int) int
	// AgeAndName ...
	AgeAndName() (int, string)
	SetAgeAndName(name string, age int)
	// TelephoneAndName ...
	GetNameAndTelephone() (name, telephone string)
	SetNameAndTelephone(name, telephone string)
}

`

	mustBeEqual(out, expected, t)
}

func TestMainNoIfaceComment(t *testing.T) {
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-D"}
	out := captureStdout(func() {
		main()
	})

	expected := `// DO NOT EDIT: Auto generated

package gen

// PersonIface ...
// Person contains data related to a person.
type PersonIface interface {
	// Name ...
	Name() string
	// SetName ...
	SetName(name string)
	// Age ...
	Age() int
	// Age ...
	SetAge(age int) int
	// AgeAndName ...
	AgeAndName() (int, string)
	SetAgeAndName(name string, age int)
	// TelephoneAndName ...
	GetNameAndTelephone() (name, telephone string)
	SetNameAndTelephone(name, telephone string)
}

`

	mustBeEqual(out, expected, t)
}

func TestMainNoCopyTypeDocs(t *testing.T) {
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-y", "PersonIface is an interface for Person."}
	out := captureStdout(func() {
		main()
	})

	expected := `// DO NOT EDIT: Auto generated

package gen

// PersonIface is an interface for Person.
type PersonIface interface {
	// Name ...
	Name() string
	// SetName ...
	SetName(name string)
	// Age ...
	Age() int
	// Age ...
	SetAge(age int) int
	// AgeAndName ...
	AgeAndName() (int, string)
	SetAgeAndName(name string, age int)
	// TelephoneAndName ...
	GetNameAndTelephone() (name, telephone string)
	SetNameAndTelephone(name, telephone string)
}

`

	mustBeEqual(out, expected, t)
}

func mustBeEqual(value, pattern string, t *testing.T) {
	if value != pattern {
		t.Fatalf("Value %s did not match expected pattern %s", value, pattern)
	}
}

// not thread safe
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
