package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/objectserver"
	"net/rpc"
	"os"
)

var (
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	objectServerHostname = flag.String("objectServerHostname", "localhost",
		"Hostname of image server")
	objectServerPortNum = flag.Uint("objectServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: objecttool [flags...] check|delete|list [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  check  hash")
	fmt.Fprintln(os.Stderr, "  get    hash baseOutputFilename")
}

type commandFunc func(objectserver.ObjectServer, []string)

type subcommand struct {
	command string
	numArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"check", 1, checkObjectSubcommand},
	{"get", 2, getObjectSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	clientName := fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum)
	client, err := rpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Printf("Error dialing\t%s\n", err)
		os.Exit(1)
	}
	objectServer := objectclient.NewObjectClient(client)
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if flag.NArg()-1 != subcommand.numArgs {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(objectServer, flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
