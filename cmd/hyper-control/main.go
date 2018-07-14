package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
)

var (
	fleetManagerHostname = flag.String("fleetManagerHostname", "",
		"Hostname of Fleet Manager")
	fleetManagerPortNum = flag.Uint("fleetManagerPortNum",
		constants.FleetManagerPortNumber,
		"Port number of Fleet Resource Manager")
	hypervisorHostname = flag.String("hypervisorHostname", "",
		"Hostname of hypervisor")
	hypervisorPortNum = flag.Uint("hypervisorPortNum",
		constants.HypervisorPortNumber, "Port number of hypervisor")
	location = flag.String("location", "",
		"Location to search for hypervisors")

	logger log.DebugLogger
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: hyper-control [flags...] command [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add-address MACaddr [IPaddr]")
	fmt.Fprintln(os.Stderr, "  add-subnet ID IPgateway IPmask DNSserver...")
	fmt.Fprintln(os.Stderr, "  get-updates")
	fmt.Fprintln(os.Stderr, "  remove-excess-addresses MaxFreeAddr")
}

type commandFunc func([]string, log.DebugLogger)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add-address", 1, 2, addAddressSubcommand},
	{"add-subnet", 4, -1, addSubnetSubcommand},
	{"get-updates", 0, 0, getUpdatesSubcommand},
	{"remove-excess-addresses", 1, 1, removeExcessAddressesSubcommand},
}

func main() {
	if err := loadflags.LoadForCli("hyper-control"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	logger = cmdlogger.New()
	if err := setupclient.SetupTls(false); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
