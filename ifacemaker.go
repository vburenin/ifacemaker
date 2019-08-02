package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/vburenin/ifacemaker/maker"
)

type cmdlineArgs struct {
	Files        []string `short:"f" long:"file" description:"Go source file to read, either filename or glob" required:"true"`
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
	var files []string
	for _, filePattern := range args.Files {
		matches, err := filepath.Glob(filePattern)
		if err != nil {
			log.Fatal(err)
		}
		files = append(files, matches...)
	}
	result, err := maker.Make(files, args.StructType, args.Comment, args.PkgName, args.IfaceName, args.IfaceComment, args.copyDocs, args.CopyTypeDoc)
	if err != nil {
		log.Fatal(err.Error())
	}

	if args.Output == "" {
		fmt.Println(string(result))
	} else {
		ioutil.WriteFile(args.Output, result, 0644)
	}
}
