package maker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"

	"golang.org/x/tools/imports"
)

// Method describes the code and documentation
// tied into a method
type Method struct {
	Code string
	Docs []string
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

// GetReceiverTypeName returns the name of the
// receiver type and the function declaration
func GetReceiverTypeName(src []byte, fl interface{}) (string, *ast.FuncDecl) {
	fd, ok := fl.(*ast.FuncDecl)
	if !ok {
		return "", nil
	}
	if fd.Recv.NumFields() != 1 {
		return "", nil
	}
	t := fd.Recv.List[0].Type
	st := string(src[t.Pos()-1 : t.End()-1])
	if len(st) > 0 && st[0] == '*' {
		st = st[1:]
	}
	return st, fd
}

// GetParameters ...
func GetParameters(src []byte, fl *ast.FieldList) ([]string, bool) {
	if fl == nil {
		return nil, false
	}
	merged := false
	parts := []string{}

	for _, l := range fl.List {
		names := make([]string, len(l.Names))
		if len(names) > 1 {
			merged = true
		}
		for i, n := range l.Names {
			names[i] = n.Name
		}

		t := string(src[l.Type.Pos()-1 : l.Type.End()-1])

		var v string
		if len(names) > 0 {
			v = fmt.Sprintf("%s %s", strings.Join(names, ", "), t)
			merged = true
		} else {
			v = t
		}
		parts = append(parts, v)
	}
	return parts, merged || len(parts) > 1
}

// FormatCode sets the options of the imports
// pkg and then applies the Process method
// which by default removes all of the imports
// not used and formats the remaining
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
func MakeInterface(comment, pkgName, ifaceName, ifaceComment string, methods []string, imports []string) ([]byte, error) {
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
		fmt.Sprintf("// %s", ifaceComment),
		fmt.Sprintf("type %s interface {", ifaceName),
	)
	output = append(output, methods...)
	output = append(output, "}")
	code := strings.Join(output, "\n")
	return FormatCode(code)
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
func ParseStruct(src []byte, structName string, copyDocs bool) (methods []Method, imports []string) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, i := range a.Imports {
		if i.Name != nil {
			imports = append(imports, fmt.Sprintf("%s %s", i.Name.String(), i.Path.Value))
		} else {
			imports = append(imports, fmt.Sprintf("%s", i.Path.Value))
		}
	}

	for _, d := range a.Decls {
		if a, fd := GetReceiverTypeName(src, d); a == structName {
			methodName := fd.Name.String()
			if methodName[0] > 'Z' {
				continue
			}
			params, _ := GetParameters(src, fd.Type.Params)
			ret, merged := GetParameters(src, fd.Type.Results)

			var retValues string
			if merged {
				retValues = fmt.Sprintf("(%s)", strings.Join(ret, ", "))
			} else {
				retValues = strings.Join(ret, ", ")
			}
			method := fmt.Sprintf("%s(%s) %s", methodName, strings.Join(params, ", "), retValues)
			var docs []string
			if fd.Doc != nil && copyDocs {
				for _, d := range fd.Doc.List {
					docs = append(docs, string(src[d.Pos()-1:d.End()-1]))
				}
			}
			methods = append(methods, Method{
				Code: method,
				Docs: docs,
			})
		}
	}
	return
}
