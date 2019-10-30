package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/hypervisor/dhcpd"
	"github.com/Cloud-Foundations/Dominator/hypervisor/httpd"
	"github.com/Cloud-Foundations/Dominator/hypervisor/manager"
	"github.com/Cloud-Foundations/Dominator/hypervisor/metadatad"
	"github.com/Cloud-Foundations/Dominator/hypervisor/rpcd"
	"github.com/Cloud-Foundations/Dominator/hypervisor/tftpbootd"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/Dominator/lib/net"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/tricorder/go/tricorder"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

var (
	dhcpServerOnBridgesOnly = flag.Bool("dhcpServerOnBridgesOnly", false,
		"If true, run the DHCP server on bridge interfaces only")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	networkBootImage = flag.String("networkBootImage", "pxelinux.0",
		"Name of boot image passed via DHCP option")
	objectCacheSize = flagutil.Size(10 << 30)
	portNum         = flag.Uint("portNum", constants.HypervisorPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	showVGA  = flag.Bool("showVGA", false, "If true, show VGA console")
	stateDir = flag.String("stateDir", "/var/lib/hypervisor",
		"Name of state directory")
	testMemoryAvailable = flag.Uint64("testMemoryAvailable", 0,
		"test if memory is allocatable and exit (units of MiB)")
	tftpbootImageStream = flag.String("tftpbootImageStream", "",
		"Name of default image stream for network booting")
	username = flag.String("username", "nobody",
		"Name of user to run VMs")
	volumeDirectories flagutil.StringList
)

func init() {
	flag.Var(&objectCacheSize, "objectCacheSize",
		"maximum size of object cache")
	flag.Var(&volumeDirectories, "volumeDirectories",
		"Comma separated list of volume directories. If empty, scan for space")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: hypervisor [flags...] [run|stop|stop-vms-on-next-stop]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
}

func processCommand(args []string) {
	if len(args) < 1 {
		return
	} else if len(args) > 1 {
		printUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "run":
		return
	case "stop":
		requestStop()
	case "stop-vms-on-next-stop":
		configureVMsToStopOnNextStop()
	default:
		printUsage()
		os.Exit(2)
	}
}

func main() {
	if err := loadflags.LoadForDaemon("hypervisor"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Usage = printUsage
	flag.Parse()
	processCommand(flag.Args())
	if *testMemoryAvailable > 0 {
		nBytes := *testMemoryAvailable << 20
		mem := make([]byte, nBytes)
		for pos := uint64(0); pos < nBytes; pos += 4096 {
			mem[pos] = 0
		}
		os.Exit(0)
	}
	tricorder.RegisterFlags()
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Must run the Hypervisor as root")
		os.Exit(1)
	}
	logger := serverlogger.New("")
	if err := setupserver.SetupTls(); err != nil {
		logger.Fatalln(err)
	}
	if err := os.MkdirAll(*stateDir, dirPerms); err != nil {
		logger.Fatalf("Cannot create state directory: %s\n", err)
	}
	bridges, bridgeMap, err := net.ListBroadcastInterfaces(
		net.InterfaceTypeBridge, logger)
	if err != nil {
		logger.Fatalf("Cannot list bridges: %s\n", err)
	}
	dhcpInterfaces := make([]string, 0, len(bridges))
	vlanIdToBridge := make(map[uint]string)
	for _, bridge := range bridges {
		if vlanId, err := net.GetBridgeVlanId(bridge.Name); err != nil {
			logger.Fatalf("Cannot get VLAN Id for bridge: %s: %s\n",
				bridge.Name, err)
		} else if vlanId < 0 {
			logger.Printf("Bridge: %s has no EtherNet port, ignoring\n",
				bridge.Name)
		} else {
			if *dhcpServerOnBridgesOnly {
				dhcpInterfaces = append(dhcpInterfaces, bridge.Name)
			}
			if !strings.HasPrefix(bridge.Name, "br@") {
				vlanIdToBridge[uint(vlanId)] = bridge.Name
				logger.Printf("Bridge: %s, VLAN Id: %d\n", bridge.Name, vlanId)
			}
		}
	}
	dhcpServer, err := dhcpd.New(dhcpInterfaces, logger)
	if err != nil {
		logger.Fatalf("Cannot start DHCP server: %s\n", err)
	}
	if err := dhcpServer.SetNetworkBootImage(*networkBootImage); err != nil {
		logger.Fatalf("Cannot set NetworkBootImage name: %s\n", err)
	}
	imageServerAddress := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	tftpbootServer, err := tftpbootd.New(imageServerAddress,
		*tftpbootImageStream, logger)
	if err != nil {
		logger.Fatalf("Cannot start tftpboot server: %s\n", err)
	}
	managerObj, err := manager.New(manager.StartOptions{
		BridgeMap:          bridgeMap,
		DhcpServer:         dhcpServer,
		ImageServerAddress: imageServerAddress,
		Logger:             logger,
		ObjectCacheBytes:   uint64(objectCacheSize),
		ShowVgaConsole:     *showVGA,
		StateDir:           *stateDir,
		Username:           *username,
		VlanIdToBridge:     vlanIdToBridge,
		VolumeDirectories:  volumeDirectories,
	})
	if err != nil {
		logger.Fatalf("Cannot start hypervisor: %s\n", err)
	}
	if err := listenForControl(managerObj, logger); err != nil {
		logger.Fatalf("Cannot listen for control: %s\n", err)
	}
	httpd.AddHtmlWriter(managerObj)
	if len(bridges) < 1 {
		logger.Println("No bridges found: entering log-only mode")
	} else {
		rpcHtmlWriter, err := rpcd.Setup(managerObj, dhcpServer, tftpbootServer,
			logger)
		if err != nil {
			logger.Fatalf("Cannot start rpcd: %s\n", err)
		}
		httpd.AddHtmlWriter(rpcHtmlWriter)
	}
	httpd.AddHtmlWriter(logger)
	err = metadatad.StartServer(*portNum, bridges, managerObj, logger)
	if err != nil {
		logger.Fatalf("Cannot start metadata server: %s\n", err)
	}
	if err := httpd.StartServer(*portNum, managerObj, false); err != nil {
		logger.Fatalf("Unable to create http server: %s\n", err)
	}
}
