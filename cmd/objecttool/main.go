package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/objectserver"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
)

var (
	chunkSize = flag.Uint("chunkSize", 65535, "Chunk size for bandwidth test")
	debug     = flag.Bool("debug", false,
		"If true, show debugging output")
	objectServerHostname = flag.String("objectServerHostname", "localhost",
		"Hostname of image server")
	objectServerPortNum = flag.Uint("objectServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	testDuration = flag.Duration("testDuration", time.Second*10,
		"Duration of bandwidth test")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: objecttool [flags...] check|delete|list [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add    files...")
	fmt.Fprintln(os.Stderr, "  check  hash")
	fmt.Fprintln(os.Stderr, "  get    hash baseOutputFilename")
	fmt.Fprintln(os.Stderr, "  mget   hashesFile directory")
	fmt.Fprintln(os.Stderr, "  test-bandwidth-from-server")
	fmt.Fprintln(os.Stderr, "  test-bandwidth-to-server")
}

type commandFunc func(objectserver.ObjectServer, []string)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add", 1, -1, addObjectsSubcommand},
	{"check", 1, 1, checkObjectSubcommand},
	{"get", 2, 2, getObjectSubcommand},
	{"mget", 2, 2, getObjectsSubcommand},
	{"test-bandwidth-from-server", 0, 0, testBandwidthFromServerSubcommand},
	{"test-bandwidth-to-server", 0, 0, testBandwidthToServerSubcommand},
}

func main() {
	if err := loadflags.LoadForCli("objecttool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	objectServer := objectclient.NewObjectClient(fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum))
	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
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
