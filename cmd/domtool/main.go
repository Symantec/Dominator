package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
)

var (
	cpuPercent = flag.Uint("cpuPercent", 0,
		"CPU speed as percentage of capacity (default 50)")
	networkSpeedPercent = flag.Uint("networkSpeedPercent",
		constants.DefaultNetworkSpeedPercent,
		"Network speed as percentage of capacity")
	scanExcludeList  flagutil.StringList = constants.ScanExcludeList
	scanSpeedPercent                     = flag.Uint("scanSpeedPercent",
		constants.DefaultScanSpeedPercent,
		"Scan speed as percentage of capacity")
	domHostname = flag.String("domHostname", "localhost",
		"Hostname of dominator")
	domPortNum = flag.Uint("domPortNum", constants.DominatorPortNumber,
		"Port number of dominator")

	dominatorSrpcClient *srpc.Client
)

func init() {
	flag.Var(&scanExcludeList, "scanExcludeList",
		"Comma separated list of patterns to exclude from scanning")
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: domtool [flags...] command")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"clear-safety-shutoff", "sub", 1, 1, clearSafetyShutoffSubcommand},
	{"configure-subs", "", 0, 0, configureSubsSubcommand},
	{"disable-updates", "reason", 1, 1, disableUpdatesSubcommand},
	{"enable-updates", "reason", 1, 1, enableUpdatesSubcommand},
	{"get-default-image", "", 0, 0, getDefaultImageSubcommand},
	{"get-subs-configuration", "", 0, 0, getSubsConfigurationSubcommand},
	{"set-default-image", "", 1, 1, setDefaultImageSubcommand},
}

func getClient() *srpc.Client {
	return dominatorSrpcClient
}

func doMain() int {
	if err := loadflags.LoadForCli("domtool"); err != nil {
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
	clientName := fmt.Sprintf("%s:%d", *domHostname, *domPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
		os.Exit(1)
	}
	dominatorSrpcClient = client
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
