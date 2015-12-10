package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/objectclient"
	"net/rpc"
	"os"
	"path"
)

var (
	certFile = flag.String("certFile",
		path.Join(os.Getenv("HOME"), ".ssl/cert.pem"),
		"Name of file containing the user SSL certificate")
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	keyFile = flag.String("keyFile",
		path.Join(os.Getenv("HOME"), ".ssl/key.pem"),
		"Name of file containing the user SSL key")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: imagetool [flags...] add|check|delete|list [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add    name imagefile filterfile triggerfile")
	fmt.Fprintln(os.Stderr, "  check  name")
	fmt.Fprintln(os.Stderr, "  delete name")
	fmt.Fprintln(os.Stderr, "  diffi  tool image image")
	fmt.Fprintln(os.Stderr, "  diffs  tool image sub")
	fmt.Fprintln(os.Stderr, "  get    name directory")
	fmt.Fprintln(os.Stderr, "  list")
	fmt.Fprintln(os.Stderr, "  show   name")
}

type commandFunc func(*rpc.Client, *objectclient.ObjectClient, []string)

type subcommand struct {
	command string
	numArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add", 4, addImageSubcommand},
	{"check", 1, checkImageSubcommand},
	{"delete", 1, deleteImageSubcommand},
	{"diffi", 3, diffImageVImageSubcommand},
	{"diffs", 3, diffImageVSubSubcommand},
	{"get", 2, getImageSubcommand},
	{"list", 0, listImagesSubcommand},
	{"show", 1, showImageSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	setupTls()
	clientName := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	imageClient, err := rpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
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
