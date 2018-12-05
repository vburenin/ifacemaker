package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/vburenin/ifacemaker/maker"
)

type cmdlineArgs struct {
	Files        []string `short:"f" long:"file" description:"Go source file to read" required:"true"`
	StructType   string   `short:"s" long:"struct" description:"Generate an interface for this structure name" required:"true"`
	IfaceName    string   `short:"i" long:"iface" description:"Name of the generated interface" required:"true"`
	PkgName      string   `short:"p" long:"pkg" description:"Package name for the generated interface" required:"true"`
	IfaceComment string   `short:"y" long:"iface-comment" description:"Comment for the interface, default is '// <iface> ...'"`

	// jessevdk/go-flags doesn't support default values for boolean flags,
	// so we use a string for backwards-compatibility and then convert it to a bool later.
	CopyDocs string `short:"d" long:"doc" description:"Copy docs from methods" option:"true" option:"false" default:"true"`
	copyDocs bool

	CopyTypeDoc bool   `short:"D" long:"type-doc" description:"Copy type doc from struct"`
	Comment     string `short:"c" long:"comment" description:"Append comment to top"`
	Output      string `short:"o" long:"output" description:"Output file name. If not provided, result will be printed to stdout."`
}

func run(args cmdlineArgs) {
	allMethods := []string{}
	allImports := []string{}
	mset := make(map[string]struct{})
	iset := make(map[string]struct{})
	var typeDoc string
	for _, f := range args.Files {
		src, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err.Error())
		}
		methods, imports, parsedTypeDoc := maker.ParseStruct(src, args.StructType, args.copyDocs, args.CopyTypeDoc)
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
		if typeDoc == "" {
			typeDoc = parsedTypeDoc
		}
	}

	if typeDoc != "" {
		args.IfaceComment = fmt.Sprintf("%s\n%s", args.IfaceComment, typeDoc)
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
	var args cmdlineArgs
	_, err := flags.ParseArgs(&args, os.Args)
	if err != nil {
		if flags.WroteHelp(err) {
			return
		}
		// No need to log the error, flags.ParseArgs() already does this
		os.Exit(1)
	}

	// Workaround because jessevdk/go-flags doesn't support default values for boolean flags
	args.copyDocs = args.CopyDocs == "true"

	if args.IfaceComment == "" {
		args.IfaceComment = fmt.Sprintf("%s ...", args.IfaceName)
	}

	run(args)
}
