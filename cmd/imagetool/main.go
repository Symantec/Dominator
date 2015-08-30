package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/objectclient"
	"net/rpc"
	"os"
)

var (
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: imagetool [flags...] add|check|delete|list [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add    name imagefile filterfile")
	fmt.Fprintln(os.Stderr, "  check  name")
	fmt.Fprintln(os.Stderr, "  delete name")
	fmt.Fprintln(os.Stderr, "  list")
}

type commandFunc func(*rpc.Client, *objectclient.ObjectClient, []string)

type subcommand struct {
	command string
	numArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add", 3, addImageSubcommand},
	{"check", 1, checkImageSubcommand},
	{"delete", 1, deleteImageSubcommand},
	{"list", 0, listImagesSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	clientName := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	imageClient, err := rpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Printf("Error dialing\t%s\n", err)
		os.Exit(1)
	}
	objectClient := objectclient.NewObjectClient(clientName)
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if flag.NArg()-1 != subcommand.numArgs {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(imageClient, objectClient, flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
