package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/fleetmanager/httpd"
	"github.com/Symantec/Dominator/fleetmanager/hypervisors"
	"github.com/Symantec/Dominator/fleetmanager/hypervisors/fsstorer"
	"github.com/Symantec/Dominator/fleetmanager/rpcd"
	"github.com/Symantec/Dominator/fleetmanager/topology"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log/serverlogger"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/tricorder/go/tricorder"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

var (
	checkTopology = flag.Bool("checkTopology", false,
		"If true, perform a one-time check, write to stdout and exit")
	topologyCheckInterval = flag.Duration("topologyCheckInterval",
		time.Minute, "Configuration check interval")
	portNum = flag.Uint("portNum", constants.FleetManagerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/fleet-manager",
		"Name of state directory")
	topologyDir = flag.String("topologyDir", "",
		"Name of topology directory or Git repository")
)

func doCheck() {
	topo, err := topology.Load(*topologyDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.WriteWithIndent(os.Stdout, "    ", topo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func main() {
	if err := loadflags.LoadForDaemon("fleet-manager"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	if *checkTopology {
		doCheck()
	}
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	if err := setupserver.SetupTls(); err != nil {
		logger.Fatalln(err)
	}
	if err := os.MkdirAll(*stateDir, dirPerms); err != nil {
		logger.Fatalf("Cannot create state directory: %s\n", err)
	}
	topologyChannel, err := topology.Watch(*topologyDir,
		*topologyCheckInterval, logger)
	if err != nil {
		logger.Fatalf("Cannot watch for topology: %s\n", err)
	}
	storer, err := fsstorer.New(filepath.Join(*stateDir, "hypervisor-db"),
		logger)
	if err != nil {
		logger.Fatalf("Cannot create DB: %s\n", err)
	}
	hyperManager, err := hypervisors.New(storer, logger)
	if err != nil {
		logger.Fatalf("Cannot create hypervisors manager: %s\n", err)
	}
	rpcHtmlWriter, err := rpcd.Setup(hyperManager, logger)
	if err != nil {
		logger.Fatalf("Cannot start rpcd: %s\n", err)
	}
	webServer, err := httpd.StartServer(*portNum, logger)
	if err != nil {
		logger.Fatalf("Unable to create http server: %s\n", err)
	}
	webServer.AddHtmlWriter(hyperManager)
	webServer.AddHtmlWriter(rpcHtmlWriter)
	webServer.AddHtmlWriter(logger)
	for topology := range topologyChannel {
		logger.Println("Received new topology")
		webServer.UpdateTopology(topology)
		hyperManager.UpdateTopology(topology)
	}

}
