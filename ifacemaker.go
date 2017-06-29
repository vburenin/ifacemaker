package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/mkideal/cli"
	"github.com/vburenin/ifacemaker/maker"
)

type cmdlineArgs struct {
	cli.Helper
	Files      []string `cli:"*f,file" usage:"Go source file to read"`
	StructType string   `cli:"*s,struct" usage:"Generate an interface for this structure name"`
	IfaceName  string   `cli:"*i,iface" usage:"Name of the generated interface"`
	PkgName    string   `cli:"*p,pkg" usage:"Package name for the generated interface"`
	CopyDocs   bool     `cli:"d,doc" usage:"Copy docs from methods" dft:"true"`
	Output     string   `cli:"o,output" usage:"Output file name. If not provided, result will be printed to stdout."`
}

func run(args *cmdlineArgs) {
	maker := &maker.Maker{
		StructName: args.StructType,
		CopyDocs:   args.CopyDocs,
	}
	for _, f := range args.Files {
		src, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err.Error())
		}
		err = maker.ParseSource(src, filepath.Base(f))
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	result, err := maker.MakeInterface(args.PkgName, args.IfaceName)
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
		run(argv)
		return nil
	})
}
