package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
)

var (
	loggerHostname = flag.String("loggerHostname", "localhost",
		"Hostname of log server")
	loggerName    = flag.String("loggerName", "", "Name of logger")
	loggerPortNum = flag.Uint("loggerPortNum", 0, "Port number of log server")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: logtool [flags...] debug|print|set-debug-level [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  debug           level args...")
	fmt.Fprintln(os.Stderr, "  print                 args...")
	fmt.Fprintln(os.Stderr, "  set-debug-level level")
}

type commandFunc func(*srpc.Client, []string, log.Logger)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"debug", 2, -1, debugSubcommand},
	{"print", 1, -1, printSubcommand},
	{"set-debug-level", 1, 1, setDebugLevelSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	logger := cmdlogger.New()
	if err := setupclient.SetupTls(true); err != nil {
		logger.Fatalln(err)
	}
	clientName := fmt.Sprintf("%s:%d", *loggerHostname, *loggerPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		logger.Fatalf("Error dialing: %s\n", err)
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
			subcommand.cmdFunc(client, flag.Args()[1:], logger)
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
