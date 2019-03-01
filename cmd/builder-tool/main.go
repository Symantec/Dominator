// +build linux

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
)

var (
	alwaysShowBuildLog = flag.Bool("alwaysShowBuildLog", false,
		"If true, show build log even for successful builds")
	imaginatorHostname = flag.String("imaginatorHostname", "localhost",
		"Hostname of image build server")
	imaginatorPortNum = flag.Uint("imaginatorPortNum",
		constants.ImaginatorPortNumber,
		"Port number of image build server")
	expiresIn = flag.Duration("expiresIn", time.Hour,
		"How long before the image expires (auto deletes)")
	imageFilename = flag.String("imageFilename", "",
		"Name of file to write image to")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	maxSourceAge = flag.Duration("maxSourceAge", time.Hour,
		"Maximum age of a source image before it is rebuilt")

	minimumExpiration = 15 * time.Minute
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: builder-tool [flags...] command [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  build-from-manifest manifestDir [stream-name]")
	fmt.Fprintln(os.Stderr, "  build-image stream-name [git-branch]")
	fmt.Fprintln(os.Stderr, "  build-tree-from-manifest manifestDir")
	fmt.Fprintln(os.Stderr, "  process-manifest manifestDir rootDir")
}

type commandFunc func([]string, log.Logger)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"build-from-manifest", 2, 2, buildFromManifestSubcommand},
	{"build-image", 1, 2, buildImageSubcommand},
	{"build-tree-from-manifest", 1, 1, buildTreeFromManifestSubcommand},
	{"process-manifest", 2, 2, processManifestSubcommand},
}

var imaginatorSrpcClient *srpc.Client
var imageServerSrpcClient *srpc.Client

func getImaginatorClient() *srpc.Client {
	if imaginatorSrpcClient == nil {
		var err error
		clientName := fmt.Sprintf("%s:%d",
			*imaginatorHostname, *imaginatorPortNum)
		imaginatorSrpcClient, err = srpc.DialHTTP("tcp", clientName, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error dialing: %s: %s\n", clientName, err)
			os.Exit(1)
		}
	}
	return imaginatorSrpcClient
}

func getImageServerClient() *srpc.Client {
	if imageServerSrpcClient == nil {
		var err error
		clientName := fmt.Sprintf("%s:%d",
			*imageServerHostname, *imageServerPortNum)
		imageServerSrpcClient, err = srpc.DialHTTP("tcp", clientName, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error dialing: %s: %s\n", clientName, err)
			os.Exit(1)
		}
	}
	return imageServerSrpcClient
}

func main() {
	if err := loadflags.LoadForCli("builder-tool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	if *expiresIn > 0 && *expiresIn < minimumExpiration {
		fmt.Fprintf(os.Stderr, "Minimum expiration: %s\n", minimumExpiration)
		os.Exit(2)
	}
	logger := cmdlogger.New()
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Unsetenv("LANG")
	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(flag.Args()[1:], logger)
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
