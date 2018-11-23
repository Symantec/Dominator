package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/imageunpacker/httpd"
	"github.com/Symantec/Dominator/imageunpacker/rpcd"
	"github.com/Symantec/Dominator/imageunpacker/unpacker"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/log/serverlogger"
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
	portNum = flag.Uint("portNum", constants.ImageUnpackerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/image-unpacker",
		"Name of state directory.")
)

func main() {
	if err := loadflags.LoadForDaemon("image-unpacker"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	tricorder.RegisterFlags()
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Must run the Image Unpacker as root")
		os.Exit(1)
	}
	logger := serverlogger.New("")
	if err := setupserver.SetupTls(); err != nil {
		logger.Fatalln(err)
	}
	if err := os.MkdirAll(*stateDir, dirPerms); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create state directory: %s\n", err)
		os.Exit(1)
	}
	unpackerObj, err := unpacker.Load(*stateDir,
		fmt.Sprintf("%s:%d", *imageServerHostname, *imageServerPortNum), logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot start unpacker: %s\n", err)
		os.Exit(1)
	}
	rpcHtmlWriter := rpcd.Setup(unpackerObj, logger)
	httpd.AddHtmlWriter(unpackerObj)
	httpd.AddHtmlWriter(rpcHtmlWriter)
	httpd.AddHtmlWriter(logger)
	if err = httpd.StartServer(*portNum, unpackerObj, false); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server: %s\n", err)
		os.Exit(1)
	}
}
