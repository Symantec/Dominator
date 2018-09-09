package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"github.com/Symantec/Dominator/lib/tags"
)

var (
	adjacentVM = flag.String("adjacentVM", "",
		"IP address of VM adjacent (same Hypervisor) to VM being created")
	dhcpTimeout = flag.Duration("dhcpTimeout", time.Minute,
		"Time to wait before timing out on DHCP request from VM")
	fleetManagerHostname = flag.String("fleetManagerHostname", "",
		"Hostname of Fleet Manager")
	fleetManagerPortNum = flag.Uint("fleetManagerPortNum",
		constants.FleetManagerPortNumber,
		"Port number of Fleet Resource Manager")
	forceIfNotStopped = flag.Bool("forceIfNotStopped", false,
		"If true, snapshot or restore VM even if not stopped")
	hypervisorHostname = flag.String("hypervisorHostname", "",
		"Hostname of hypervisor")
	hypervisorPortNum = flag.Uint("hypervisorPortNum",
		constants.HypervisorPortNumber, "Port number of hypervisor")
	hypervisorProxy = flag.String("hypervisorProxy", "",
		"URL for hypervisor proxy")
	imageFile = flag.String("imageFile", "",
		"Name of RAW image file to boot with")
	imageName    = flag.String("imageName", "", "Name of image to boot with")
	imageTimeout = flag.Duration("imageTimeout", time.Minute,
		"Time to wait before timing out on image fetch")
	imageURL = flag.String("imageURL", "",
		"Name of URL of image to boot with")
	location = flag.String("location", "",
		"Location to search for hypervisors")
	memory       = flag.Uint64("memory", 128, "memory in MiB")
	milliCPUs    = flag.Uint("milliCPUs", 250, "milli CPUs")
	minFreeBytes = flag.Uint64("minFreeBytes", 64<<20,
		"minimum number of free bytes in root volume")
	ownerGroups  flagutil.StringList
	ownerUsers   flagutil.StringList
	probePortNum = flag.Uint("probePortNum", 0, "Port number on VM to probe")
	probeTimeout = flag.Duration("probeTimeout", time.Minute*5,
		"Time to wait before timing out on probing VM port")
	secondaryVolumeSizes flagutil.StringList
	subnetId             = flag.String("subnetId", "",
		"Subnet ID to launch VM in")
	roundupPower = flag.Uint64("roundupPower", 24,
		"power of 2 to round up root volume size")
	snapshotRootOnly = flag.Bool("snapshotRootOnly", false,
		"If true, snapshot only the root volume")
	traceMetadata = flag.Bool("traceMetadata", false,
		"If true, trace metadata calls until interrupted")
	userDataFile = flag.String("userDataFile", "",
		"Name file containing user-data accessible from the metadata server")
	vmHostname     = flag.String("vmHostname", "", "Hostname for VM")
	vmTags         tags.Tags
	volumeFilename = flag.String("volumeFilename", "",
		"Name of file to write volume data to")
	volumeIndex = flag.Uint("volumeIndex", 0, "Index of volume to get")

	logger log.DebugLogger
)

func init() {
	flag.Var(&ownerGroups, "ownerGroups", "Groups who own the VM")
	flag.Var(&ownerUsers, "ownerUsers", "Extra users who own the VM")
	flag.Var(&secondaryVolumeSizes, "secondaryVolumeSizes",
		"Sizes for secondary volumes")
	flag.Var(&vmTags, "vmTags", "Tags to apply to VM")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: vm-control [flags...] command [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  become-primary-vm-owner IPaddr")
	fmt.Fprintln(os.Stderr, "  change-vm-owner-users IPaddr")
	fmt.Fprintln(os.Stderr, "  change-vm-tags IPaddr")
	fmt.Fprintln(os.Stderr, "  create-vm")
	fmt.Fprintln(os.Stderr, "  destroy-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  discard-vm-old-image IPaddr")
	fmt.Fprintln(os.Stderr, "  discard-vm-old-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  discard-vm-snapshot IPaddr")
	fmt.Fprintln(os.Stderr, "  get-vm-info IPaddr")
	fmt.Fprintln(os.Stderr, "  get-vm-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  get-vm-volume IPaddr")
	fmt.Fprintln(os.Stderr, "  import-local-vm info-file root-volume")
	fmt.Fprintln(os.Stderr, "  import-virsh-vm MACaddr domain")
	fmt.Fprintln(os.Stderr, "  list-hypervisors")
	fmt.Fprintln(os.Stderr, "  list-locations [TopLocation]")
	fmt.Fprintln(os.Stderr, "  list-vms")
	fmt.Fprintln(os.Stderr, "  probe-vm-port IPaddr")
	fmt.Fprintln(os.Stderr, "  replace-vm-image IPaddr")
	fmt.Fprintln(os.Stderr, "  replace-vm-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  restore-vm-from-snapshot IPaddr")
	fmt.Fprintln(os.Stderr, "  restore-vm-image IPaddr")
	fmt.Fprintln(os.Stderr, "  restore-vm-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  snapshot-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  start-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  stop-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  trace-vm-metadata IPaddr")
}

type commandFunc func([]string, log.DebugLogger)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"become-primary-vm-owner", 1, 1, becomePrimaryVmOwnerSubcommand},
	{"change-vm-owner-users", 1, 1, changeVmOwnerUsersSubcommand},
	{"change-vm-tags", 1, 1, changeVmTagsSubcommand},
	{"create-vm", 0, 0, createVmSubcommand},
	{"destroy-vm", 1, 1, destroyVmSubcommand},
	{"discard-vm-old-image", 1, 1, discardVmOldImageSubcommand},
	{"discard-vm-old-user-data", 1, 1, discardVmOldUserDataSubcommand},
	{"discard-vm-snapshot", 1, 1, discardVmSnapshotSubcommand},
	{"get-vm-info", 1, 1, getVmInfoSubcommand},
	{"get-vm-user-data", 1, 1, getVmUserDataSubcommand},
	{"get-vm-volume", 1, 1, getVmVolumeSubcommand},
	{"import-local-vm", 2, 2, importLocalVmSubcommand},
	{"import-virsh-vm", 2, 2, importVirshVmSubcommand},
	{"list-hypervisors", 0, 0, listHypervisorsSubcommand},
	{"list-locations", 0, 1, listLocationsSubcommand},
	{"list-vms", 0, 0, listVMsSubcommand},
	{"probe-vm-port", 1, 1, probeVmPortSubcommand},
	{"replace-vm-image", 1, 1, replaceVmImageSubcommand},
	{"replace-vm-user-data", 1, 1, replaceVmUserDataSubcommand},
	{"restore-vm-from-snapshot", 1, 1, restoreVmFromSnapshotSubcommand},
	{"restore-vm-image", 1, 1, restoreVmImageSubcommand},
	{"restore-vm-user-data", 1, 1, restoreVmUserDataSubcommand},
	{"snapshot-vm", 1, 1, snapshotVmSubcommand},
	{"start-vm", 1, 1, startVmSubcommand},
	{"stop-vm", 1, 1, stopVmSubcommand},
	{"trace-vm-metadata", 1, 1, traceVmMetadataSubcommand},
}

func main() {
	if err := loadflags.LoadForCli("vm-control"); err != nil {
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
