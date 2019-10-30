// +build linux

package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/imagebuilder/builder"
	"github.com/Cloud-Foundations/Dominator/imagebuilder/httpd"
	"github.com/Cloud-Foundations/Dominator/imagebuilder/rpcd"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupserver"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

var (
	configurationUrl = flag.String("configurationUrl",
		"file:///etc/imaginator/conf.json", "URL containing configuration")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	imageRebuildInterval = flag.Duration("imageRebuildInterval", time.Hour,
		"time between automatic rebuilds of images")
	portNum = flag.Uint("portNum", constants.ImaginatorPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	slaveDriverConfigurationFile = flag.String("slaveDriverConfigurationFile",
		"", "Name of configuration file for slave builders")
	stateDir = flag.String("stateDir", "/var/lib/imaginator",
		"Name of state directory")
	variablesFile = flag.String("variablesFile", "",
		"A JSON encoded file containing special variables (i.e. secrets)")
)

func main() {
	if err := loadflags.LoadForDaemon("imaginator"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	tricorder.RegisterFlags()
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Must run the Image Builder as root")
		os.Exit(1)
	}
	logger := serverlogger.New("")
	if umask := syscall.Umask(022); umask != 022 {
		// Since we can't cleanly fix umask for all threads, fail instead.
		logger.Fatalf("Umask must be 022, not 0%o\n", umask)
	}
	if err := setupserver.SetupTls(); err != nil {
		logger.Fatalln(err)
	}
	if err := os.MkdirAll(*stateDir, dirPerms); err != nil {
		logger.Fatalf("Cannot create state directory: %s\n", err)
	}
	slaveDriver, err := createSlaveDriver(logger)
	if err != nil {
		logger.Fatalf("Error starting slave driver: %s\n", err)
	}
	builderObj, err := builder.Load(*configurationUrl, *variablesFile,
		*stateDir,
		fmt.Sprintf("%s:%d", *imageServerHostname, *imageServerPortNum),
		*imageRebuildInterval, slaveDriver, logger)
	if err != nil {
		logger.Fatalf("Cannot start builder: %s\n", err)
	}
	rpcHtmlWriter, err := rpcd.Setup(builderObj, logger)
	if err != nil {
		logger.Fatalf("Cannot start builder: %s\n", err)
	}
	httpd.AddHtmlWriter(builderObj)
	if slaveDriver != nil {
		httpd.AddHtmlWriter(slaveDriver)
	}
	httpd.AddHtmlWriter(rpcHtmlWriter)
	httpd.AddHtmlWriter(logger)
	if err = httpd.StartServer(*portNum, builderObj, false); err != nil {
		logger.Fatalf("Unable to create http server: %s\n", err)
	}
}
