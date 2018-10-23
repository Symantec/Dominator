package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"github.com/Symantec/Dominator/lib/tags"
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
	hypervisorTags      tags.Tags
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	installerImageStream = flag.String("installerImageStream", "",
		"Name of default image stream for building bootable installer ISO")
	location = flag.String("location", "",
		"Location to search for hypervisors")
	offerTimeout = flag.Duration("offerTimeout", time.Minute+time.Second,
		"How long to offer DHCP OFFERs and ACKs")
	netbootFiles        tags.Tags
	netbootFilesTimeout = flag.Duration("netbootFilesTimeout",
		time.Minute+time.Second, "How long to provide extra files via TFTP")
	netbootTimeout = flag.Duration("netbootTimeout", time.Minute,
		"Time to wait for DHCP ACKs to be sent")
	numAcknowledgementsToWaitFor = flag.Uint("numAcknowledgementsToWaitFor",
		2, "Number of DHCP ACKs to wait for")
	storageLayoutFilename = flag.String("storageLayoutFilename", "",
		"Name of file containing storage layout for installing machine")
	topologyDir = flag.String("topologyDir", "",
		"Name of local topology directory in Git repository")

	logger log.DebugLogger
)

func init() {
	flag.Var(&hypervisorTags, "hypervisorTags", "Tags to apply to Hypervisor")
	flag.Var(&netbootFiles, "netbootFiles", "Extra files served by TFTP server")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: hyper-control [flags...] command [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add-address MACaddr [IPaddr]")
	fmt.Fprintln(os.Stderr, "  add-subnet ID IPgateway IPmask DNSserver...")
	fmt.Fprintln(os.Stderr, "  change-tags")
	fmt.Fprintln(os.Stderr, "  get-machine-info hostname")
	fmt.Fprintln(os.Stderr, "  get-updates")
	fmt.Fprintln(os.Stderr, "  make-installer-iso hostname dirname")
	fmt.Fprintln(os.Stderr, "  netboot-host hostname")
	fmt.Fprintln(os.Stderr, "  netboot-machine MACaddr IPaddr [hostname]")
	fmt.Fprintln(os.Stderr, "  remove-excess-addresses MaxFreeAddr")
	fmt.Fprintln(os.Stderr, "  rollout-image name")
	fmt.Fprintln(os.Stderr, "  write-netboot-files hostname dirname")
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
	{"change-tags", 0, 0, changeTagsSubcommand},
	{"get-machine-info", 1, 1, getMachineInfoSubcommand},
	{"get-updates", 0, 0, getUpdatesSubcommand},
	{"make-installer-iso", 2, 2, makeInstallerIsoSubcommand},
	{"netboot-host", 1, 1, netbootHostSubcommand},
	{"netboot-machine", 2, 3, netbootMachineSubcommand},
	{"remove-excess-addresses", 1, 1, removeExcessAddressesSubcommand},
	{"rollout-image", 1, 1, rolloutImageSubcommand},
	{"write-netboot-files", 2, 2, writeNetbootFilesSubcommand},
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
