package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"os"
	"time"
)

var (
	computedFilesRoot = flag.String("computedFilesRoot", "",
		"Name of directory tree containing computed files")
	debug             = flag.Bool("debug", false, "Enable debug mode")
	deleteBeforeFetch = flag.Bool("deleteBeforeFetch", false,
		"If true, delete prior to Fetch rather than during Update")
	file = flag.String("file", "",
		"Name of file to write encoded data to")
	filterFile = flag.String("filterFile", "",
		"Replacement filter file to apply when pushing image")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	interval = flag.Uint("interval", 1,
		"Seconds to sleep between Polls")
	networkSpeedPercent = flag.Uint("networkSpeedPercent",
		constants.DefaultNetworkSpeedPercent,
		"Network speed as percentage of capacity")
	newConnection = flag.Bool("newConnection", false,
		"If true, (re)open a connection for each Poll")
	numPolls = flag.Int("numPolls", 1,
		"The number of polls to run (infinite: < 0)")
	objectServerHostname = flag.String("objectServerHostname", "localhost",
		"Hostname of image server")
	objectServerPortNum = flag.Uint("objectServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	scanExcludeList  flagutil.StringList = constants.ScanExcludeList
	scanSpeedPercent                     = flag.Uint("scanSpeedPercent",
		constants.DefaultScanSpeedPercent,
		"Scan speed as percentage of capacity")
	shortPoll = flag.Bool("shortPoll", false,
		"If true, perform a short poll which does not request image or object data")
	showTimes = flag.Bool("showTimes", false,
		"If true, show time taken for some operations")
	subHostname = flag.String("subHostname", "localhost", "Hostname of sub")
	subPortNum  = flag.Uint("subPortNum", constants.SubPortNumber,
		"Port number of sub")
	timeout = flag.Duration("timeout", 15*time.Minute,
		"timeout for long operations")
	triggersFile = flag.String("triggersFile", "",
		"Replacement triggers file to apply when pushing image")
	triggersString = flag.String("triggersString", "",
		"Replacement triggers string to apply when pushing image (ignored if triggersFile is set)")
	wait = flag.Uint("wait", 0, "Seconds to sleep after last Poll")

	timeoutTime time.Time
)

func init() {
	flag.Var(&scanExcludeList, "scanExcludeList",
		"Comma separated list of patterns to exclude from scanning")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: subtool [flags...] fetch|get-config|poll|set-config")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  cleanup")
	fmt.Fprintln(os.Stderr, "  fetch hashesFile")
	fmt.Fprintln(os.Stderr, "  get-config")
	fmt.Fprintln(os.Stderr, "  get-file remoteFile localFile")
	fmt.Fprintln(os.Stderr, "  poll")
	fmt.Fprintln(os.Stderr, "  push-file source dest")
	fmt.Fprintln(os.Stderr, "  push-image image")
	fmt.Fprintln(os.Stderr, "  set-config")
	fmt.Fprintln(os.Stderr, "  show-update-request image")
	fmt.Fprintln(os.Stderr, "  wait-for-image image")
}

func getSubClient() *srpc.Client {
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, time.Second*5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing %s: %s\n", clientName, err)
		os.Exit(3)
	}
	return client
}

func getSubClientRetry() *srpc.Client {
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	var client *srpc.Client
	var err error
	for time.Now().Before(timeoutTime) {
		client, err = srpc.DialHTTP("tcp", clientName, time.Second*5)
		if err == nil {
			return client
		}
		if err == srpc.ErrorMissingCertificate ||
			err == srpc.ErrorBadCertificate ||
			err == srpc.ErrorAccessToMethodDenied {
			// Never going to happen. Bail out.
			fmt.Fprintf(os.Stderr, "Error dialing %s: %s\n", clientName, err)
			os.Exit(3)
		}
	}
	fmt.Fprintf(os.Stderr, "Error dialing %s: %s\n", clientName, err)
	os.Exit(3)
	return nil
}

type getSubClientFunc func() *srpc.Client
type commandFunc func(getSubClientFunc, []string)

type subcommand struct {
	command      string
	numArgs      int
	getSubClient getSubClientFunc
	cmdFunc      commandFunc
}

var subcommands = []subcommand{
	{"cleanup", 0, getSubClient, cleanupSubcommand},
	{"fetch", 1, getSubClient, fetchSubcommand},
	{"get-config", 0, getSubClient, getConfigSubcommand},
	{"get-file", 2, getSubClient, getFileSubcommand},
	{"poll", 0, getSubClient, pollSubcommand},
	{"push-file", 2, getSubClient, pushFileSubcommand},
	{"push-image", 1, getSubClientRetry, pushImageSubcommand},
	{"set-config", 0, getSubClient, setConfigSubcommand},
	{"show-update-request", 1, getSubClientRetry, showUpdateRequestSubcommand},
	{"wait-for-image", 1, getSubClientRetry, waitForImageSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	if *triggersFile != "" && *triggersString != "" {
		fmt.Fprintln(os.Stderr,
			"Cannot specify both -triggersFile and -triggersString")
		os.Exit(2)
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	timeoutTime = time.Now().Add(*timeout)
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if flag.NArg()-1 != subcommand.numArgs {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(subcommand.getSubClient, flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
