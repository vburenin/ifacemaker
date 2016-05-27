package main

import (
	"io/ioutil"
	"log"

	"fmt"

	"github.com/mkideal/cli"
	"github.com/vburenin/ifacemaker/maker"
)

type cmdlineArgs struct {
	cli.Helper
	Files      []string `cli:"*f,file" usage:"go source file with the structure"`
	StructType string   `cli:"*s,struct" usage:"Structure type name to look for"`
	IfaceName  string   `cli:"*i,iface" usage:"Exported interface name"`
	NoDoc      bool     `cli:"d,nodoc" usage:"Copy docs from methods" dft:"false"`
	Output     string   `cli:"o,output" usage:"Output file name. If not provided, result will be printed to stdout"`
}

func run(args *cmdlineArgs) {
	allMethods := []string{}
	mset := map[string]struct{}{}
	for _, f := range args.Files {
		src, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err.Error())
		}
		for _, m := range maker.ParseStruct(src, args.StructType, args.NoDoc) {
			if _, ok := mset[m]; !ok {
				allMethods = append(allMethods, m)
				mset[m] = struct{}{}
			}
		}
	}

	result, err := maker.MakeInterface(args.IfaceName, allMethods, nil)
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
