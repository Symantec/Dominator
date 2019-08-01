package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/net/rrdialer"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"github.com/Symantec/Dominator/lib/tags"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

var (
	adjacentVM = flag.String("adjacentVM", "",
		"IP address of VM adjacent (same Hypervisor) to VM being created")
	consoleType       hyper_proto.ConsoleType
	destroyProtection = flag.Bool("destroyProtection", false,
		"If true, do not destroy running VM")
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
	includeUnhealthy = flag.Bool("includeUnhealthy", false,
		"If true, list connected but unhealthy hypervisors")
	imageFile = flag.String("imageFile", "",
		"Name of RAW image file to boot with")
	imageName    = flag.String("imageName", "", "Name of image to boot with")
	imageTimeout = flag.Duration("imageTimeout", time.Minute,
		"Time to wait before timing out on image fetch")
	imageURL = flag.String("imageURL", "",
		"Name of URL of image to boot with")
	localVmCreate = flag.String("localVmCreate", "",
		"Command to make local VM when exporting. The VM name is given as the argument. The VM JSON is available on stdin")
	localVmDestroy = flag.String("localVmDestroy", "",
		"Command to destroy local VM when exporting. The VM name is given as the argument")
	location = flag.String("location", "",
		"Location to search for hypervisors")
	memory       flagutil.Size
	milliCPUs    = flag.Uint("milliCPUs", 0, "milli CPUs (default 250)")
	minFreeBytes = flagutil.Size(256 << 20)
	ownerGroups  flagutil.StringList
	ownerUsers   flagutil.StringList
	probePortNum = flag.Uint("probePortNum", 0, "Port number on VM to probe")
	probeTimeout = flag.Duration("probeTimeout", time.Minute*5,
		"Time to wait before timing out on probing VM port")
	secondarySubnetIDs   flagutil.StringList
	secondaryVolumeSizes flagutil.StringList
	serialPort           = flag.Uint("serialPort", 0,
		"Serial port number on VM")
	skipBootloader = flag.Bool("skipBootloader", false,
		"If true, directly boot into the kernel")
	subnetId = flag.String("subnetId", "",
		"Subnet ID to launch VM in")
	requestIPs   flagutil.StringList
	roundupPower = flag.Uint64("roundupPower", 28,
		"power of 2 to round up root volume size")
	snapshotRootOnly = flag.Bool("snapshotRootOnly", false,
		"If true, snapshot only the root volume")
	traceMetadata = flag.Bool("traceMetadata", false,
		"If true, trace metadata calls until interrupted")
	userDataFile = flag.String("userDataFile", "",
		"Name file containing user-data accessible from the metadata server")
	vmHostname = flag.String("vmHostname", "", "Hostname for VM")
	vmTags     tags.Tags
	vncViewer  = flag.String("vncViewer", defaultVncViewer,
		"Path to VNC viewer")
	volumeFilename = flag.String("volumeFilename", "",
		"Name of file to write volume data to")
	volumeIndex = flag.Uint("volumeIndex", 0,
		"Index of volume to get or delete")

	logger   log.DebugLogger
	rrDialer *rrdialer.Dialer
)

func init() {
	flag.Var(&consoleType, "consoleType",
		"type of graphical console (default none)")
	flag.Var(&memory, "memory", "memory (default 1GiB)")
	flag.Var(&minFreeBytes, "minFreeBytes",
		"minimum number of free bytes in root volume")
	flag.Var(&ownerGroups, "ownerGroups", "Groups who own the VM")
	flag.Var(&ownerUsers, "ownerUsers", "Extra users who own the VM")
	flag.Var(&requestIPs, "requestIPs", "Request specific IPs, if available")
	flag.Var(&secondarySubnetIDs, "secondarySubnetIDs", "Secondary Subnet IDs")
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
	fmt.Fprintln(os.Stderr, "  change-vm-console-type IPaddr")
	fmt.Fprintln(os.Stderr, "  change-vm-destroy-protection IPaddr")
	fmt.Fprintln(os.Stderr, "  change-vm-owner-users IPaddr")
	fmt.Fprintln(os.Stderr, "  change-vm-tags IPaddr")
	fmt.Fprintln(os.Stderr, "  connect-to-vm-console IPaddr")
	fmt.Fprintln(os.Stderr, "  connect-to-vm-serial-port IPaddr")
	fmt.Fprintln(os.Stderr, "  copy-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  create-vm")
	fmt.Fprintln(os.Stderr, "  delete-vm-volume IPaddr")
	fmt.Fprintln(os.Stderr, "  destroy-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  discard-vm-old-image IPaddr")
	fmt.Fprintln(os.Stderr, "  discard-vm-old-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  discard-vm-snapshot IPaddr")
	fmt.Fprintln(os.Stderr, "  export-local-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  export-virsh-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  get-vm-info IPaddr")
	fmt.Fprintln(os.Stderr, "  get-vm-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  get-vm-volume IPaddr")
	fmt.Fprintln(os.Stderr, "  import-local-vm info-file root-volume")
	fmt.Fprintln(os.Stderr, "  import-virsh-vm MACaddr domain [[MAC IP]...]")
	fmt.Fprintln(os.Stderr, "  list-hypervisors")
	fmt.Fprintln(os.Stderr, "  list-locations [TopLocation]")
	fmt.Fprintln(os.Stderr, "  list-vms")
	fmt.Fprintln(os.Stderr, "  migrate-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  patch-vm-image IPaddr")
	fmt.Fprintln(os.Stderr, "  probe-vm-port IPaddr")
	fmt.Fprintln(os.Stderr, "  replace-vm-image IPaddr")
	fmt.Fprintln(os.Stderr, "  replace-vm-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  restore-vm-from-snapshot IPaddr")
	fmt.Fprintln(os.Stderr, "  restore-vm-image IPaddr")
	fmt.Fprintln(os.Stderr, "  restore-vm-user-data IPaddr")
	fmt.Fprintln(os.Stderr, "  set-vm-migrating IPaddr")
	fmt.Fprintln(os.Stderr, "  snapshot-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  start-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  stop-vm IPaddr")
	fmt.Fprintln(os.Stderr, "  trace-vm-metadata IPaddr")
	fmt.Fprintln(os.Stderr, "  unset-vm-migrating IPaddr")
}

type commandFunc func([]string, log.DebugLogger) error

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"become-primary-vm-owner", 1, 1, becomePrimaryVmOwnerSubcommand},
	{"change-vm-console-type", 1, 1, changeVmConsoleTypeSubcommand},
	{"change-vm-destroy-protection", 1, 1, changeVmDestroyProtectionSubcommand},
	{"change-vm-owner-users", 1, 1, changeVmOwnerUsersSubcommand},
	{"change-vm-tags", 1, 1, changeVmTagsSubcommand},
	{"connect-to-vm-console", 1, 1, connectToVmConsoleSubcommand},
	{"connect-to-vm-serial-port", 1, 1, connectToVmSerialPortSubcommand},
	{"copy-vm", 1, 1, copyVmSubcommand},
	{"create-vm", 0, 0, createVmSubcommand},
	{"delete-vm-volume", 1, 1, deleteVmVolumeSubcommand},
	{"destroy-vm", 1, 1, destroyVmSubcommand},
	{"discard-vm-old-image", 1, 1, discardVmOldImageSubcommand},
	{"discard-vm-old-user-data", 1, 1, discardVmOldUserDataSubcommand},
	{"discard-vm-snapshot", 1, 1, discardVmSnapshotSubcommand},
	{"export-local-vm", 1, 1, exportLocalVmSubcommand},
	{"export-virsh-vm", 1, 1, exportVirshVmSubcommand},
	{"get-vm-info", 1, 1, getVmInfoSubcommand},
	{"get-vm-user-data", 1, 1, getVmUserDataSubcommand},
	{"get-vm-volume", 1, 1, getVmVolumeSubcommand},
	{"import-local-vm", 2, 2, importLocalVmSubcommand},
	{"import-virsh-vm", 2, -1, importVirshVmSubcommand},
	{"list-hypervisors", 0, 0, listHypervisorsSubcommand},
	{"list-locations", 0, 1, listLocationsSubcommand},
	{"list-vms", 0, 0, listVMsSubcommand},
	{"migrate-vm", 1, 1, migrateVmSubcommand},
	{"patch-vm-image", 1, 1, patchVmImageSubcommand},
	{"probe-vm-port", 1, 1, probeVmPortSubcommand},
	{"replace-vm-image", 1, 1, replaceVmImageSubcommand},
	{"replace-vm-user-data", 1, 1, replaceVmUserDataSubcommand},
	{"restore-vm-from-snapshot", 1, 1, restoreVmFromSnapshotSubcommand},
	{"restore-vm-image", 1, 1, restoreVmImageSubcommand},
	{"restore-vm-user-data", 1, 1, restoreVmUserDataSubcommand},
	{"set-vm-migrating", 1, 1, setVmMigratingSubcommand},
	{"snapshot-vm", 1, 1, snapshotVmSubcommand},
	{"start-vm", 1, 1, startVmSubcommand},
	{"stop-vm", 1, 1, stopVmSubcommand},
	{"trace-vm-metadata", 1, 1, traceVmMetadataSubcommand},
	{"unset-vm-migrating", 1, 1, unsetVmMigratingSubcommand},
}

func doMain() int {
	if err := loadflags.LoadForCli("vm-control"); err != nil {
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
	if err := setupclient.SetupTls(false); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var err error
	rrDialer, err = rrdialer.New(&net.Dialer{Timeout: time.Second * 10}, "",
		logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rrDialer.WaitForBackgroundResults(time.Second)
	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
				printUsage()
				return 2
			}
			if err := subcommand.cmdFunc(flag.Args()[1:], logger); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			return 0
		}
	}
	printUsage()
	return 2
}

func main() {
	os.Exit(doMain())
}
