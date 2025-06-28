package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var testBinary string

var src = `package main

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
	return p.age
}

// Age ...
func (p *Person) SetAge(age int) int {
	p.age = age
	return p.age
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
}

`

var src2 = `package maker

import (
	"github.com/vburenin/ifacemaker/maker/footest"
)

type TestImpl struct{}

func (s *TestImpl) GetUser(userID string) *footest.User {
	return &footest.User{}
}

func (s *TestImpl) CreateUser(user *footest.User) (*footest.User, error) {
	return &footest.User{}, nil
}

func (s *TestImpl) fooHelper() string {
	return ""
}`

var src2_extend = `package maker
import (
	"github.com/vburenin/ifacemaker/maker/footest"
)

func (s *TestImpl) UpdateUser(userID string) *footest.User {
    return &footest.User{}, nil
}
`
var src3 = `package footest

type User struct {
	ID   string
	Name string
}`

var src4 = `package footest

// Hammer is in the same package but in a different file.
type Smiter struct {
	options Options
}
func (s *Smiter) Smite(weapon Hammer) error {
	return nil
}
`

var src5 = `package bartest

import (
	"github.com/test/footest"
)

type Healer struct {
	options Options
}
func (h *Healer) Heal(smiter *footest.Smiter) error {
	return nil
}
func (h *Healer) Buff(smiter *footest.Smiter, buffs []*footest.BuffType) error {
	return nil
}
`
var src6 = `package bazztest
// ParentStruct ...
type ParentStruct struct {}

// DoSomething does something
func (ps *ParentStruct) DoSomething() error {
	return nil
}

// ChildStruct ...
type ChildStruct struct {
	ParentStruct
}
`

var srcFile = filepath.Join(os.TempDir(), "ifacemaker_src.go")
var srcFile2 = filepath.Join(os.TempDir(), "test_impl.go")
var srcFile2_ext = filepath.Join(os.TempDir(), "test_impl_extended.go")
var srcFile3 = filepath.Join(os.TempDir(), "footest", "footest.go")
var srcFile4 = filepath.Join(os.TempDir(), "footest", "smiter.go")
var srcFile5 = filepath.Join(os.TempDir(), "bartest", "healer.go")
var srcFile6 = filepath.Join(os.TempDir(), "bazztest", "custom_structs.go")

func TestMain(m *testing.M) {
	testBinary = os.Args[0]
	// Add /tmp/footest directory
	dirPath := filepath.Join(os.TempDir(), "footest")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.Mkdir(dirPath, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("Failed to create directory: %s", err))
		}
	}
	// Add /tmp/bartest directory
	dirPath = filepath.Join(os.TempDir(), "bartest")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.Mkdir(dirPath, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("Failed to create directory: %s", err))
		}
	}

	// Add /tmp/bazztest directory
	dirPath = filepath.Join(os.TempDir(), "bazztest")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.Mkdir(dirPath, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("Failed to create directory: %s", err))
		}
	}
	writeTestSourceFile(src, srcFile)
	writeTestSourceFile(src2, srcFile2)
	writeTestSourceFile(src2_extend, srcFile2_ext)
	writeTestSourceFile(src3, srcFile3)
	writeTestSourceFile(src4, srcFile4)
	writeTestSourceFile(src5, srcFile5)
	writeTestSourceFile(src6, srcFile6)

	os.Exit(m.Run())
}

func writeTestSourceFile(src, path string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("Failed to open test source file: %s", err))
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

	require.Equal(t, expected, out)
}

func TestMainNoIfaceComment(t *testing.T) {
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
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-D"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)

}

func TestMainNoCopyTypeDocs(t *testing.T) {
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
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-y", "PersonIface is an interface for Person."}
	out := captureStdout(func() {
		main()
	})
	require.Equal(t, expected, out)

}

func TestMainNoCopyMethodDocs(t *testing.T) {
	expected := `// DO NOT EDIT: Auto generated

package gen

// PersonIface ...
type PersonIface interface {
	Name() string
	SetName(name string)
	Age() int
	SetAge(age int) int
	AgeAndName() (int, string)
	SetAgeAndName(name string, age int)
	GetNameAndTelephone() (name, telephone string)
	SetNameAndTelephone(name, telephone string)
}

`
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-d=false"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)

}

func TestMainDoNotImportPackageName(t *testing.T) {
	expected := `// DO NOT EDIT: Auto generated

package footest

// TestInterface ...
type TestInterface interface {
	GetUser(userID string) *User
	CreateUser(user *User) (*User, error)
}

`
	os.Args = []string{"cmd", "-f", srcFile2, "-s", "TestImpl", "-p", "footest", "-c", "DO NOT EDIT: Auto generated", "-i", "TestInterface", "-d=false"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)
}

func TestMainFileGlob(t *testing.T) {
	expected := `// DO NOT EDIT: Auto generated

package footest

// TestInterface ...
type TestInterface interface {
	GetUser(userID string) *User
	CreateUser(user *User) (*User, error)
}

`
	os.Args = []string{"cmd", "-f", srcFile2, "-s", "TestImpl", "-p", "footest", "-c", "DO NOT EDIT: Auto generated", "-i", "TestInterface", "-d=false"}
	out := captureStdout(func() {
		main()
	})
	require.Equal(t, expected, out)

}

func TestMainDefaultComment(t *testing.T) {
	expected := `// Code generated by ifacemaker; DO NOT EDIT.

package footest

// TestInterface ...
type TestInterface interface {
	GetUser(userID string) *User
	CreateUser(user *User) (*User, error)
}

`
	os.Args = []string{"cmd", "-f", srcFile2, "-s", "TestImpl", "-p", "footest", "-i", "TestInterface", "-d=false"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)
}

func TestMainUsingUnknownDeclarationInSamePackage(t *testing.T) {
	expected := `// DO NOT EDIT: Auto generated

package another

import (
	. "github.com/test/footest"
)

// Smiter ...
type Smiter interface {
	Smite(weapon Hammer) error
}

`
	os.Args = []string{"cmd", "-f", srcFile4, "-m", "github.com/test/footest", "-s", "Smiter", "-i", "Smiter", "-p", "another", "-c", "DO NOT EDIT: Auto generated", "-d=false"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)
}

func TestMainUsingImportedDeclaration(t *testing.T) {
	expected := `// Code generated by ifacemaker; DO NOT EDIT.

package gen

import (
	"github.com/test/footest"
)

// Healer ...
type Healer interface {
	Heal(smiter *footest.Smiter) error
	Buff(smiter *footest.Smiter, buffs []*footest.BuffType) error
}

`
	os.Args = []string{"cmd", "-f", srcFile5, "-s", "Healer", "-i", "Healer", "-p", "gen"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)
}

func TestMainWithMultipleFiles(t *testing.T) {
	expected := `// Code generated by ifacemaker; DO NOT EDIT.

package gen

import (
	"github.com/vburenin/ifacemaker/maker/footest"
)

// Test ...
type Test interface {
	GetUser(userID string) *footest.User
	CreateUser(user *footest.User) (*footest.User, error)
	UpdateUser(userID string) *footest.User
}

`
	os.Args = []string{"cmd", "-f", srcFile2, "-f", srcFile2_ext, "-s", "TestImpl", "-i", "Test", "-p", "gen"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)
}

func TestMainWithPromoted(t *testing.T) {
	expected := `// Code generated by ifacemaker; DO NOT EDIT.

package gen

// Test ...
type Test interface {
	// DoSomething does something
	DoSomething() error
}

`
	os.Args = []string{"cmd", "-f", srcFile6, "-s", "ChildStruct", "-i", "Test", "-p", "gen", "-P"}
	out := captureStdout(func() {
		main()
	})

	require.Equal(t, expected, out)
}

func TestMainWriteToFile(t *testing.T) {
	outPath := filepath.Join(os.TempDir(), "ifacemaker_out.go")
	os.Remove(outPath)

	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-y", "PersonIface is an interface for Person.", "-D"}
	expected := captureStdout(func() { main() })

	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-y", "PersonIface is an interface for Person.", "-D", "-o", outPath}
	main()

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(data)))
}

func TestMainParseArgsError(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		os.Args = []string{"cmd", "-f"}
		main()
		return
	}
	cmd := exec.Command(testBinary, "-test.run=TestMainParseArgsError")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return
	}
	t.Fatalf("main did not exit as expected")
}

func TestMainGlobError(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		os.Args = []string{"cmd", "-f", "[", "-s", "Person", "-i", "Iface", "-p", "gen"}
		main()
		return
	}
	cmd := exec.Command(testBinary, "-test.run=TestMainGlobError")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return
	}
	t.Fatalf("main did not exit as expected")
}

func TestMainHelp(t *testing.T) {
	if os.Getenv("BE_CRASHER_HELP") == "1" {
		os.Args = []string{"cmd", "-h"}
		main()
		return
	}
	cmd := exec.Command(testBinary, "-test.run=TestMainHelp")
	cmd.Env = append(os.Environ(), "BE_CRASHER_HELP=1")
	err := cmd.Run()
	require.NoError(t, err)
}

func TestMainMakeError(t *testing.T) {
	if os.Getenv("BE_CRASHER_MAKEERR") == "1" {
		os.Args = []string{"cmd", "-f", "/no/such/file.go", "-s", "Foo", "-p", "gen", "-i", "Iface"}
		main()
		return
	}
	cmd := exec.Command(testBinary, "-test.run=TestMainMakeError")
	cmd.Env = append(os.Environ(), "BE_CRASHER_MAKEERR=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return
	}
	t.Fatalf("main did not exit as expected")
}

func TestMainOutputCreateError(t *testing.T) {
	if os.Getenv("BE_CRASHER_OUTERR") == "1" {
		os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-i", "Iface", "-o", "/nonexistent/path/out.go"}
		main()
		return
	}
	cmd := exec.Command(testBinary, "-test.run=TestMainOutputCreateError")
	cmd.Env = append(os.Environ(), "BE_CRASHER_OUTERR=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return
	}
	t.Fatalf("main did not exit as expected")
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
	if _, err := io.Copy(&buf, r); err != nil {
		fmt.Printf("error:[%v] copying file", err)
		return ""
	}
	return buf.String()
}
func TestMainHelpDirect(t *testing.T) {
	os.Args = []string{"cmd", "-h"}
	out := captureStdout(func() { main() })
	require.Contains(t, out, "Usage:")
}

func TestMainOutputWriteError(t *testing.T) {
	if os.Getenv("BE_CRASHER_WRITEERR") == "1" {
		os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-i", "Iface", "-p", "gen", "-o", "/dev/full"}
		main()
		return
	}
	cmd := exec.Command(testBinary, "-test.run=TestMainOutputWriteError")
	cmd.Env = append(os.Environ(), "BE_CRASHER_WRITEERR=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
		return
	}
	t.Fatalf("main did not exit as expected")
}
