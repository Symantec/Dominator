package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"log"
	"os"
	"time"
)

var (
	amiName   = flag.String("amiName", "", "AMI Name property")
	expiresIn = flag.Duration("expiresIn", time.Hour,
		"Date to set for the ExpiresAt tag")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of imageserver")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber, "Port number of imageserver")
	minFreeBytes = flag.Uint64("minFreeBytes", 1<<28,
		"minimum number of free bytes in image")
	skipFile = flag.String("skipFile", "",
		"JSON encoded file containing targets to skip")
	skipString = flag.String("skipString", "",
		"JSON encoded string containing targets to skip")
	tagsFile = flag.String("tagsFile", "",
		"JSON encoded file containing tags to apply to AMIs")
	targetAccounts flagutil.StringList
	targetRegions  flagutil.StringList
)

func init() {
	flag.Var(&targetAccounts, "targetAccounts",
		"List of target account profile names")
	flag.Var(&targetRegions, "targetRegions",
		"List of target regions (default all regions)")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: ami-publisher [flags...] publish [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  delete results-file...")
	fmt.Fprintln(os.Stderr, "  expire")
	fmt.Fprintln(os.Stderr, "  publish stream-name image-leaf-name")
}

type commandFunc func([]string, *log.Logger)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"delete", 1, -1, deleteSubcommand},
	{"expire", 0, 0, expireSubcommand},
	{"publish", 2, 2, publishSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	if err := setupclient.SetupTls(true); err != nil {
		logger.Println(err)
		os.Exit(1)
	}
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
