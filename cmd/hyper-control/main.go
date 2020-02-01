package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/net/rrdialer"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

var (
	externalLeaseHostnames flagutil.StringList
	externalLeaseAddresses proto.AddressList
	fleetManagerHostname   = flag.String("fleetManagerHostname", "",
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
	installerPortNum = flag.Uint("installerPortNum",
		constants.InstallerPortNumber, "Port number of installer")
	location = flag.String("location", "",
		"Location to search for hypervisors")
	offerTimeout = flag.Duration("offerTimeout", time.Minute+time.Second,
		"How long to offer DHCP OFFERs and ACKs")
	memory              = flagutil.Size(4 << 30)
	netbootFiles        tags.Tags
	netbootFilesTimeout = flag.Duration("netbootFilesTimeout",
		time.Minute+time.Second,
		"How long to provide files via TFTP after last DHCP ACK")
	netbootTimeout = flag.Duration("netbootTimeout", time.Minute,
		"Time to wait for DHCP ACKs to be sent")
	networkInterfacesFile = flag.String("networkInterfacesFile", "",
		"File containing network interfaces for show-network-configuration")
	numAcknowledgementsToWaitFor = flag.Uint("numAcknowledgementsToWaitFor",
		2, "Number of DHCP ACKs to wait for")
	randomSeedBytes = flag.Uint("randomSeedBytes", 0,
		"Number of bytes of random seed data to inject into installing machine")
	storageLayoutFilename = flag.String("storageLayoutFilename", "",
		"Name of file containing storage layout for installing machine")
	subnetIDs       flagutil.StringList
	targetImageName = flag.String("targetImageName", "",
		"Name of image to install for netboot-{host,vm}")
	topologyDir = flag.String("topologyDir", "",
		"Name of local topology directory in Git repository")
	useKexec = flag.Bool("useKexec", false,
		"If true, use kexec to reboot into newly installed OS")
	vncViewer   = flag.String("vncViewer", "", "Path to VNC viewer for VM")
	volumeSizes = flagutil.SizeList{16 << 30}

	rrDialer *rrdialer.Dialer
)

func init() {
	flag.Var(&externalLeaseAddresses, "externalLeaseAddresses",
		"List of addresses for register-external-leases")
	flag.Var(&externalLeaseHostnames, "externalLeaseHostnames",
		"Optional list of hostnames for register-external-leases")
	flag.Var(&hypervisorTags, "hypervisorTags", "Tags to apply to Hypervisor")
	flag.Var(&memory, "memory", "memory for VM")
	flag.Var(&netbootFiles, "netbootFiles", "Extra files served by TFTP server")
	flag.Var(&subnetIDs, "subnetIDs", "Subnet IDs for VM")
	flag.Var(&volumeSizes, "volumeSizes", "Sizes for volumes for VM")
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: hyper-control [flags...] command [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"add-address", "MACaddr [IPaddr]", 1, 2, addAddressSubcommand},
	{"add-subnet", "ID IPgateway IPmask DNSserver...", 4, -1,
		addSubnetSubcommand},
	{"change-tags", "", 0, 0, changeTagsSubcommand},
	{"get-machine-info", "hostname", 1, 1, getMachineInfoSubcommand},
	{"get-updates", "", 0, 0, getUpdatesSubcommand},
	{"installer-shell", "hostname", 1, 1, installerShellSubcommand},
	{"make-installer-iso", "hostname dirname", 2, 2,
		makeInstallerIsoSubcommand},
	{"move-ip-address", "IPaddr", 1, 1, moveIpAddressSubcommand},
	{"netboot-host", "hostname", 1, 1, netbootHostSubcommand},
	{"netboot-machine", "MACaddr IPaddr [hostname]", 2, 3,
		netbootMachineSubcommand},
	{"netboot-vm", "", 0, 0, netbootVmSubcommand},
	{"power-off", "", 0, 0, powerOffSubcommand},
	{"power-on", "", 0, 0, powerOnSubcommand},
	{"register-external-leases", "", 0, 0, registerExternalLeasesSubcommand},
	{"reinstall", "", 0, 0, reinstallSubcommand},
	{"remove-excess-addresses", "MaxFreeAddr", 1, 1,
		removeExcessAddressesSubcommand},
	{"remove-ip-address", "IPaddr", 1, 1, removeIpAddressSubcommand},
	{"remove-mac-address", "MACaddr", 1, 1, removeMacAddressSubcommand},
	{"rollout-image", "name", 1, 1, rolloutImageSubcommand},
	{"show-network-configuration", "", 0, 0,
		showNetworkConfigurationSubcommand},
	{"update-network-configuration", "", 0, 0,
		updateNetworkConfigurationSubcommand},
	{"write-netboot-files", "hostname dirname", 2, 2,
		writeNetbootFilesSubcommand},
}

func loadCerts() error {
	err := setupclient.SetupTls(false)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if os.Geteuid() != 0 {
		return err
	}
	cert, e := tls.LoadX509KeyPair("/etc/ssl/hypervisor/cert.pem",
		"/etc/ssl/hypervisor/key.pem")
	if e != nil {
		return err
	}
	srpc.RegisterClientTlsConfig(&tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		Certificates:       []tls.Certificate{cert},
	})
	return nil
}

func doMain() int {
	if err := loadflags.LoadForCli("hyper-control"); err != nil {
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
	if err := loadCerts(); err != nil {
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
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
