package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
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

	objectServer objectserver.ObjectServer
)

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: objecttool [flags...] check|delete|list [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"add", "   files...", 1, -1, addObjectsSubcommand},
	{"check", " hash", 1, 1, checkObjectSubcommand},
	{"get", "   hash baseOutputFilename", 2, 2, getObjectSubcommand},
	{"mget", "  hashesFile directory", 2, 2, getObjectsSubcommand},
	{"test-bandwidth-from-server", "", 0, 0, testBandwidthFromServerSubcommand},
	{"test-bandwidth-to-server", "", 0, 0, testBandwidthToServerSubcommand},
}

func getObjectServer() objectserver.ObjectServer {
	return objectServer
}

func doMain() int {
	if err := loadflags.LoadForCli("objecttool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 2
	}
	logger := cmdlogger.New()
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	objectServer = objectclient.NewObjectClient(fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum))
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
