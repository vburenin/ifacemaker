package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	assert "github.com/stretchr/testify/assert"
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

var srcFile = os.TempDir() + "/ifacemaker_src.go"
var srcFile2 = os.TempDir() + "/test_impl.go"
var srcFile2_ext = os.TempDir() + "/test_impl_extended.go"
var srcFile3 = os.TempDir() + "/footest/footest.go"

func TestMain(m *testing.M) {
	dirPath := os.TempDir() + "/footest"
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

	assert.Equal(t, expected, out)
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

	assert.Equal(t, expected, out)
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

	assert.Equal(t, expected, out)
}

func TestMainNoCopyMethodDocs(t *testing.T) {
	os.Args = []string{"cmd", "-f", srcFile, "-s", "Person", "-p", "gen", "-c", "DO NOT EDIT: Auto generated", "-i", "PersonIface", "-d=false"}
	out := captureStdout(func() {
		main()
	})

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

	assert.Equal(t, expected, out)
}

func TestMainDoNotImportPackageName(t *testing.T) {
	os.Args = []string{"cmd", "-f", srcFile2, "-s", "TestImpl", "-p", "footest", "-c", "DO NOT EDIT: Auto generated", "-i", "TestInterface", "-d=false"}
	out := captureStdout(func() {
		main()
	})

	expected := `// DO NOT EDIT: Auto generated

package footest

// TestInterface ...
type TestInterface interface {
	GetUser(userID string) *User
	CreateUser(user *User) (*User, error)
}

`

	assert.Equal(t, expected, out)
}

func TestMainFileGlob(t *testing.T) {
	src := strings.Replace(srcFile2, "test_impl.go", "test*.go", 1)
	//assert.True(t, strings.HasSuffix(srcFile2, "test*.go"))
	//assert.Equal(t, "bla", src)
	//ssert.Contains(t, src, "test*")
	os.Args = []string{"cmd", "-f", src, "-s", "TestImpl", "-p", "footest", "-c", "DO NOT EDIT: Auto generated", "-i", "TestInterface", "-d=false"}
	out := captureStdout(func() {
		main()
	})

	expected := `// DO NOT EDIT: Auto generated

package footest

// TestInterface ...
type TestInterface interface {
	GetUser(userID string) *User
	CreateUser(user *User) (*User, error)
	UpdateUser(userID string) *User
}

`

	assert.Equal(t, expected, out)
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
