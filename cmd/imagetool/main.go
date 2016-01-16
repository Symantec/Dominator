package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
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
	fmt.Fprintln(os.Stderr, "  diff   tool left right")
	fmt.Fprintln(os.Stderr, "         left & right are image sources. Format:")
	fmt.Fprintln(os.Stderr, "         type:name where type is one of:")
	fmt.Fprintln(os.Stderr, "           f: name of file containing an image")
	fmt.Fprintln(os.Stderr, "           i: name of an image on the imageserver")
	fmt.Fprintln(os.Stderr, "           s: name of sub to poll")
	fmt.Fprintln(os.Stderr, "  get    name directory")
	fmt.Fprintln(os.Stderr, "  list")
	fmt.Fprintln(os.Stderr, "  show   name")
}

type commandFunc func([]string)

type subcommand struct {
	command string
	numArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add", 4, addImageSubcommand},
	{"check", 1, checkImageSubcommand},
	{"delete", 1, deleteImageSubcommand},
	{"diff", 3, diffSubcommand},
	{"get", 2, getImageSubcommand},
	{"list", 0, listImagesSubcommand},
	{"show", 1, showImageSubcommand},
}

var imageRpcClient *rpc.Client
var imageSrpcClient *srpc.Client
var theObjectClient *objectclient.ObjectClient

func getClients() (*rpc.Client, *srpc.Client, *objectclient.ObjectClient) {
	if imageRpcClient == nil {
		var err error
		clientName := fmt.Sprintf("%s:%d",
			*imageServerHostname, *imageServerPortNum)
		imageRpcClient, err = rpc.DialHTTP("tcp", clientName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
			os.Exit(1)
		}
		imageSrpcClient, err = srpc.DialHTTP("tcp", clientName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
			os.Exit(1)
		}
		theObjectClient = objectclient.NewObjectClient(clientName)
	}
	return imageRpcClient, imageSrpcClient, theObjectClient
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	setupTls(*certFile, *keyFile)
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if flag.NArg()-1 != subcommand.numArgs {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
