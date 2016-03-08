package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filegen"
	"github.com/Symantec/Dominator/lib/filegen/httpd"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/tricorder/go/tricorder"
	"log"
	"os"
)

var (
	caFile = flag.String("CAfile", "/etc/ssl/CA.pem",
		"Name of file containing the root of trust")
	certFile = flag.String("certFile", "/etc/ssl/filegen-server/cert.pem",
		"Name of file containing the SSL certificate")
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	keyFile = flag.String("keyFile", "/etc/ssl/filegen-server/key.pem",
		"Name of file containing the SSL key")
	portNum = flag.Uint("portNum", constants.BasicFileGenServerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: filegen-server [flags...] directory...")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "directory: tree of source files")
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	tricorder.RegisterFlags()
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Do not run the filegen server as root")
		os.Exit(1)
	}
	setupTls(*caFile, *certFile, *keyFile)
	circularBuffer := logbuf.New(*logbufLines)
	logger := log.New(circularBuffer, "", log.LstdFlags)
	manager := filegen.New(logger)
	httpd.AddHtmlWriter(manager)
	httpd.AddHtmlWriter(circularBuffer)
	for _, pathname := range flag.Args() {
		if err := registerSourceDirectory(manager, pathname, "/"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := httpd.StartServer(*portNum, manager, false); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
}
