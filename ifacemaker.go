package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/mkideal/cli"
	"github.com/vburenin/ifacemaker/maker"
)

type cmdlineArgs struct {
	cli.Helper
	Files        []string `cli:"*f,file" usage:"Go source file to read"`
	StructType   string   `cli:"*s,struct" usage:"Generate an interface for this structure name"`
	IfaceName    string   `cli:"*i,iface" usage:"Name of the generated interface"`
	PkgName      string   `cli:"*p,pkg" usage:"Package name for the generated interface"`
	IfaceComment string   `cli:"y,iface-comment" usage:"Comment for the interface, default is '// <iface> ...'"`
	CopyDocs     bool     `cli:"d,doc" usage:"Copy docs from methods" dft:"true"`
	Comment      string   `cli:"c,comment" usage:"Append comment to top"`
	Output       string   `cli:"o,output" usage:"Output file name. If not provided, result will be printed to stdout."`
}

func run(args *cmdlineArgs) {
	allMethods := []string{}
	allImports := []string{}
	mset := make(map[string]struct{})
	iset := make(map[string]struct{})
	for _, f := range args.Files {
		src, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err.Error())
		}
		methods, imports := maker.ParseStruct(src, args.StructType, args.CopyDocs)
		for _, m := range methods {
			if _, ok := mset[m.Code]; !ok {
				allMethods = append(allMethods, m.Lines()...)
				mset[m.Code] = struct{}{}
			}
		}
		for _, i := range imports {
			if _, ok := iset[i]; !ok {
				allImports = append(allImports, i)
				iset[i] = struct{}{}
			}
		}
	}

	result, err := maker.MakeInterface(args.Comment, args.PkgName, args.IfaceName, args.IfaceComment, allMethods, allImports)
	if err != nil {
		log.Fatal(err.Error())
	}

	if args.Output == "" {
		fmt.Println(string(result))
	} else {
		ioutil.WriteFile(args.Output, result, 0644)
	}
}

func main() {
	cli.Run(&cmdlineArgs{}, func(ctx *cli.Context) error {
		argv := ctx.Argv().(*cmdlineArgs)
		if argv.IfaceComment == "" {
			argv.IfaceComment = fmt.Sprintf("%s ...", argv.IfaceName)
		}
		run(argv)
		return nil
	})
}
