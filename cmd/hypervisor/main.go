package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/hypervisor/dhcpd"
	"github.com/Symantec/Dominator/hypervisor/httpd"
	"github.com/Symantec/Dominator/hypervisor/manager"
	"github.com/Symantec/Dominator/hypervisor/metadatad"
	"github.com/Symantec/Dominator/hypervisor/rpcd"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/log/serverlogger"
	"github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/tricorder/go/tricorder"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

var (
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	portNum = flag.Uint("portNum", constants.HypervisorPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	showVGA  = flag.Bool("showVGA", false, "If true, show VGA console")
	stateDir = flag.String("stateDir", "/var/lib/hypervisor",
		"Name of state directory")
	testMemoryAvailable = flag.Uint64("testMemoryAvailable", 0,
		"test if memory is allocatable and exit (units of MiB)")
	username = flag.String("username", "nobody",
		"Name of user to run VMs")
	volumeDirectories flagutil.StringList
)

func init() {
	flag.Var(&volumeDirectories, "volumeDirectories",
		"Comma separated list of volume directories. If empty, scan for space")
}

func main() {
	flag.Parse()
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
	bridges, err := net.ListBridges()
	if err != nil {
		logger.Fatalf("Cannot list bridges: %s\n", err)
	}
	bridgeNames := make([]string, 0, len(bridges))
	vlanIdToBridge := make(map[uint]string)
	for _, bridge := range bridges {
		if vlanId, err := net.GetBridgeVlanId(bridge.Name); err != nil {
			logger.Fatalf("Cannot get VLAN Id for bridge: %s: %s\n",
				bridge.Name, err)
		} else if vlanId < 0 {
			logger.Printf("Bridge: %s has no EtherNet port, ignoring\n",
				bridge.Name)
		} else {
			bridgeNames = append(bridgeNames, bridge.Name)
			vlanIdToBridge[uint(vlanId)] = bridge.Name
			logger.Printf("Bridge: %s, VLAN Id: %d\n", bridge.Name, vlanId)
		}
	}
	dhcpServer, err := dhcpd.New(bridgeNames, logger)
	if err != nil {
		logger.Fatalf("Cannot start DHCP server: %s\n", err)
	}
	managerObj, err := manager.New(manager.StartOptions{
		ImageServerAddress: fmt.Sprintf("%s:%d",
			*imageServerHostname, *imageServerPortNum),
		DhcpServer:        dhcpServer,
		Logger:            logger,
		ShowVgaConsole:    *showVGA,
		StateDir:          *stateDir,
		Username:          *username,
		VlanIdToBridge:    vlanIdToBridge,
		VolumeDirectories: volumeDirectories,
	})
	if err != nil {
		logger.Fatalf("Cannot start hypervisor: %s\n", err)
	}
	httpd.AddHtmlWriter(managerObj)
	if len(bridges) < 1 {
		logger.Println("No bridges found: entering log-only mode")
	} else {
		rpcHtmlWriter, err := rpcd.Setup(managerObj, logger)
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
