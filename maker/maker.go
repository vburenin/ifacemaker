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
		} else {
			v = t
		}
		parts = append(parts, v)

		//log.Println(reflect.TypeOf(l.Type).String())
	}
	return parts, merged || len(parts) > 1
}

func FormatCode(code string) ([]byte, error) {
	opts := &imports.Options{
		TabIndent: true,
		TabWidth:  2,
		Fragment:  true,
		Comments:  true,
	}
	return imports.Process("", []byte(code), opts)
}

func MakeInterface(pkgName, ifaceName string, methods []string, imports []string) ([]byte, error) {
	output := []string{
		"package " + pkgName,
		"import (",
	}
	output = append(output, imports...)
	output = append(output,
		")",
		fmt.Sprintf("type %s interface {", ifaceName),
	)
	output = append(output, methods...)
	output = append(output, "}")
	return FormatCode(strings.Join(output, "\n"))
}

func ParseStruct(src []byte, structName string, copyDocs bool) (methods []string, imports []string) {
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
			if fd.Doc != nil && copyDocs {
				for _, d := range fd.Doc.List {
					methods = append(methods, string(src[d.Pos()-1:d.End()-1]))
				}
			}
			methods = append(methods, method)
		}
	}
	return
}
