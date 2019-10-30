package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/net/rrdialer"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
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
	installerPortNum = flag.Uint("installerPortNum",
		constants.InstallerPortNumber, "Port number of installer")
	location = flag.String("location", "",
		"Location to search for hypervisors")
	offerTimeout = flag.Duration("offerTimeout", time.Minute+time.Second,
		"How long to offer DHCP OFFERs and ACKs")
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
	topologyDir = flag.String("topologyDir", "",
		"Name of local topology directory in Git repository")
	useKexec = flag.Bool("useKexec", false,
		"If true, use kexec to reboot into newly installed OS")

	rrDialer *rrdialer.Dialer
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
	fmt.Fprintln(os.Stderr, "  installer-shell hostname")
	fmt.Fprintln(os.Stderr, "  make-installer-iso hostname dirname")
	fmt.Fprintln(os.Stderr, "  move-ip-address IPaddr")
	fmt.Fprintln(os.Stderr, "  netboot-host hostname")
	fmt.Fprintln(os.Stderr, "  netboot-machine MACaddr IPaddr [hostname]")
	fmt.Fprintln(os.Stderr, "  reinstall")
	fmt.Fprintln(os.Stderr, "  remove-excess-addresses MaxFreeAddr")
	fmt.Fprintln(os.Stderr, "  remove-ip-address IPaddr")
	fmt.Fprintln(os.Stderr, "  remove-mac-address MACaddr")
	fmt.Fprintln(os.Stderr, "  rollout-image name")
	fmt.Fprintln(os.Stderr, "  show-network-configuration")
	fmt.Fprintln(os.Stderr, "  update-network-configuration")
	fmt.Fprintln(os.Stderr, "  write-netboot-files hostname dirname")
}

type commandFunc func([]string, log.DebugLogger) error

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
	{"installer-shell", 1, 1, installerShellSubcommand},
	{"make-installer-iso", 2, 2, makeInstallerIsoSubcommand},
	{"move-ip-address", 1, 1, moveIpAddressSubcommand},
	{"netboot-host", 1, 1, netbootHostSubcommand},
	{"netboot-machine", 2, 3, netbootMachineSubcommand},
	{"reinstall", 0, 0, reinstallSubcommand},
	{"remove-excess-addresses", 1, 1, removeExcessAddressesSubcommand},
	{"remove-ip-address", 1, 1, removeIpAddressSubcommand},
	{"remove-mac-address", 1, 1, removeMacAddressSubcommand},
	{"rollout-image", 1, 1, rolloutImageSubcommand},
	{"show-network-configuration", 0, 0, showNetworkConfigurationSubcommand},
	{"update-network-configuration", 0, 0,
		updateNetworkConfigurationSubcommand},
	{"write-netboot-files", 2, 2, writeNetbootFilesSubcommand},
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
