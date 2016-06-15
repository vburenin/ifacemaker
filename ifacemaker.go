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
	Files      []string `cli:"*f,file" usage:"Go source file to read"`
	StructType string   `cli:"*s,struct" usage:"Generate an interface for this structure name"`
	IfaceName  string   `cli:"*i,iface" usage:"Name of the generated interface"`
	PkgName    string   `cli:"*p,pkg" usage:"Package name for the generated interface"`
	CopyDocs   bool     `cli:"d,doc" usage:"Copy docs from methods" dft:"true"`
	Output     string   `cli:"o,output" usage:"Output file name. If not provided, result will be printed to stdout."`
}

func run(args *cmdlineArgs) {
	allMethods := []string{}
	mset := make(map[string]struct{})
	for _, f := range args.Files {
		src, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err.Error())
		}
		for _, m := range maker.ParseStruct(src, args.StructType, args.CopyDocs) {
			if _, ok := mset[m]; !ok {
				allMethods = append(allMethods, m)
				mset[m] = struct{}{}
			}
		}
	}

	result, err := maker.MakeInterface(args.PkgName, args.IfaceName, allMethods, nil)
	result = append(result, '\n')
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
