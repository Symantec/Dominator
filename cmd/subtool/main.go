package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/log/debuglogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
)

var (
	computedFilesRoot = flag.String("computedFilesRoot", "",
		"Name of directory tree containing computed files")
	connectTimeout = flag.Duration("connectTimeout", 15*time.Second,
		"connection timeout")
	cpuPercent = flag.Uint("cpuPercent", 0,
		"CPU speed as percentage of capacity (default 50)")
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

	logger      *debuglogger.Logger
	timeoutTime time.Time
)

func init() {
	flag.Var(&scanExcludeList, "scanExcludeList",
		"Comma separated list of patterns to exclude from scanning")
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w,
		"Usage: subtool [flags...] fetch|get-config|poll|set-config")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

func getSubClient(logger log.DebugLogger) *srpc.Client {
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, *connectTimeout)
	if err != nil {
		logger.Fatalf("Error dialing %s: %s\n", clientName, err)
	}
	return client
}

func getSubClientRetry(logger log.DebugLogger) *srpc.Client {
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
			logger.Fatalf("Error dialing %s: %s\n", clientName, err)
		}
	}
	logger.Fatalf("Error dialing %s: %s\n", clientName, err)
	return nil
}

var subcommands = []commands.Command{
	{"boost-cpu-limit", "", 0, 0, boostCpuLimitSubcommand},
	{"cleanup", "", 0, 0, cleanupSubcommand},
	{"delete", "pathname...", 1, 1, deleteSubcommand},
	{"fetch", "hashesFile", 1, 1, fetchSubcommand},
	{"get-config", "", 0, 0, getConfigSubcommand},
	{"get-file", "remoteFile localFile", 2, 2, getFileSubcommand},
	{"list-missing-objects", "image", 1, 1, listMissingObjectsSubcommand},
	{"poll", "", 0, 0, pollSubcommand},
	{"push-file", "source dest", 2, 2, pushFileSubcommand},
	{"push-image", "image", 1, 1, pushImageSubcommand},
	{"push-missing-objects", "image", 1, 1, pushMissingObjectsSubcommand},
	{"restart-service", "name", 1, 1, restartServiceSubcommand},
	{"set-config", "", 0, 0, setConfigSubcommand},
	{"show-update-request", "image", 1, 1, showUpdateRequestSubcommand},
	{"wait-for-image", "image", 1, 1, waitForImageSubcommand},
}

func doMain() int {
	if err := loadflags.LoadForCli("subtool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 2
	}
	logger = cmdlogger.New()
	if *triggersFile != "" && *triggersString != "" {
		logger.Fatalln(os.Stderr,
			"Cannot specify both -triggersFile and -triggersString")
	}
	if err := setupclient.SetupTls(true); err != nil {
		logger.Fatalln(os.Stderr, err)
	}
	timeoutTime = time.Now().Add(*timeout)
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
