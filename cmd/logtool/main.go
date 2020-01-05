package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
)

var (
	excludeRegex = flag.String("excludeRegex", "",
		"The exclude regular expression to filter out when watching (after include)")
	includeRegex = flag.String("includeRegex", "",
		"The include regular expression to filter for when watching")
	loggerHostname = flag.String("loggerHostname", "localhost",
		"Hostname of log server")
	loggerName    = flag.String("loggerName", "", "Name of logger")
	loggerPortNum = flag.Uint("loggerPortNum", 0, "Port number of log server")
)

func printSubcommands(subcommands []subcommand) {
	for _, subcommand := range subcommands {
		if subcommand.args == "" {
			fmt.Fprintln(os.Stderr, " ", subcommand.command)
		} else {
			fmt.Fprintln(os.Stderr, " ", subcommand.command, subcommand.args)
		}
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: logtool [flags...] debug|print|set-debug-level [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	printSubcommands(subcommands)
}

type commandFunc func(clients []*srpc.Client, addrs, args []string,
	logger log.Logger)

type subcommand struct {
	command          string
	args             string
	minArgs          int
	maxArgs          int
	allowMultiClient bool
	cmdFunc          commandFunc
}

var subcommands = []subcommand{
	{"debug", "          level args...", 2, -1, false, debugSubcommand},
	{"print", "                args...", 1, -1, false, printSubcommand},
	{"set-debug-level", "level", 1, 1, false, setDebugLevelSubcommand},
	{"watch", "          level", 1, 1, true, watchSubcommand},
}

func dialAll(addrs []string) ([]*srpc.Client, error) {
	clients := make([]*srpc.Client, 0, len(addrs))
	for _, addr := range addrs {
		clientName := fmt.Sprintf("%s:%d", addr, *loggerPortNum)
		if client, err := srpc.DialHTTP("tcp", clientName, 0); err != nil {
			return nil, err
		} else {
			clients = append(clients, client)
		}
	}
	return clients, nil
}

func main() {
	if err := loadflags.LoadForCli("logtool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	addrs, err := net.LookupHost(*loggerHostname)
	if err != nil {
		logger.Fatalln(err)
	}
	if len(addrs) < 1 {
		logger.Fatalf("no addresses for: %s\n", *loggerHostname)
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
			if len(addrs) > 1 && !subcommand.allowMultiClient {
				logger.Fatalf("%s does not support multiple endpoints\n",
					flag.Arg(0))
			}
			clients, err := dialAll(addrs)
			if err != nil {
				logger.Fatalln(err)
			}
			subcommand.cmdFunc(clients, addrs, flag.Args()[1:], logger)
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
