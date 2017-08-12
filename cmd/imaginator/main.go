package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/imagebuilder/builder"
	"github.com/Symantec/Dominator/imagebuilder/httpd"
	"github.com/Symantec/Dominator/imagebuilder/rpcd"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/tricorder/go/tricorder"
	"log"
	"os"
	"syscall"
	"time"
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
	stateDir = flag.String("stateDir", "/var/lib/imaginator",
		"Name of state directory")
	variablesFile = flag.String("variablesFile", "",
		"A JSON encoded file containing special variables (i.e. secrets)")
)

func main() {
	flag.Parse()
	tricorder.RegisterFlags()
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Must run the Image Builder as root")
		os.Exit(1)
	}
	circularBuffer := logbuf.New()
	logger := log.New(circularBuffer, "", log.LstdFlags)
	if err := setupserver.SetupTls(); err != nil {
		logger.Println(err)
		circularBuffer.Flush()
		os.Exit(1)
	}
	if err := os.MkdirAll(*stateDir, dirPerms); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create state directory: %s\n", err)
		os.Exit(1)
	}
	builderObj, err := builder.Load(*configurationUrl, *variablesFile,
		*stateDir,
		fmt.Sprintf("%s:%d", *imageServerHostname, *imageServerPortNum),
		*imageRebuildInterval, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot start builder: %s\n", err)
		os.Exit(1)
	}
	rpcHtmlWriter, err := rpcd.Setup(builderObj, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot start builder: %s\n", err)
		os.Exit(1)
	}
	httpd.AddHtmlWriter(builderObj)
	httpd.AddHtmlWriter(rpcHtmlWriter)
	httpd.AddHtmlWriter(circularBuffer)
	if err = httpd.StartServer(*portNum, builderObj, false); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server: %s\n", err)
		os.Exit(1)
	}
}
