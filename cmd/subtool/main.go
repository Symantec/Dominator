package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"net/rpc"
	"os"
	"strings"
)

var (
	debug               = flag.Bool("debug", false, "Enable debug mode")
	file                = flag.String("file", "", "Name of file to write encoded data to")
	interval            = flag.Uint("interval", 1, "Seconds to sleep between Polls")
	networkSpeedPercent = flag.Uint("networkSpeedPercent", 10,
		"Network speed as percentage of capacity")
	newConnection = flag.Bool("newConnection", false,
		"If true, (re)open a connection for each Poll")
	numPolls = flag.Int("numPolls", 1,
		"The number of polls to run (infinite: < 0)")
	scanExcludeList = flag.String("scanExcludeList",
		strings.Join(constants.ScanExcludeList, ","),
		"Comma separated list of patterns to exclude from scanning")
	scanSpeedPercent = flag.Uint("scanSpeedPercent", 2,
		"Scan speed as percentage of capacity")
	subHostname = flag.String("subHostname", "localhost", "Hostname of sub")
	subPortNum  = flag.Uint("subPortNum", constants.SubPortNumber,
		"Port number of sub")
	wait = flag.Uint("wait", 0, "Seconds to sleep after last Poll")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: subtool [flags...] get-config|poll|set-config")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  get-config")
	fmt.Fprintln(os.Stderr, "  poll")
	fmt.Fprintln(os.Stderr, "  set-config")
}

type commandFunc func(*rpc.Client, []string)

type subcommand struct {
	command string
	numArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"get-config", 0, getConfigSubcommand},
	{"poll", 0, pollSubcommand},
	{"set-config", 0, setConfigSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	client, err := rpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
		os.Exit(1)
	}
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if flag.NArg()-1 != subcommand.numArgs {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(client, flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
