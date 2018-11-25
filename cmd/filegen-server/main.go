package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filegen"
	"github.com/Symantec/Dominator/lib/filegen/httpd"
	"github.com/Symantec/Dominator/lib/filegen/util"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log/serverlogger"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/tricorder/go/tricorder"
)

var (
	configFile = flag.String("configFile", "/var/lib/filegen-server/config",
		"Name of file containing the configuration")
	permitInsecureMode = flag.Bool("permitInsecureMode", false,
		"If true, run in insecure mode. This gives remote access to all")
	portNum = flag.Uint("portNum", constants.BasicFileGenServerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
)

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: filegen-server [flags...] directory...")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "directory: tree of source files")
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Do not run the filegen server as root")
		os.Exit(1)
	}
	if err := loadflags.LoadForDaemon("filegen-server"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Usage = printUsage
	flag.Parse()
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	if err := setupserver.SetupTls(); err != nil {
		if *permitInsecureMode {
			logger.Println(err)
		} else {
			logger.Fatalln(err)
		}
	}
	manager := filegen.New(logger)
	if *configFile != "" {
		if err := util.LoadConfiguration(manager, *configFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		ch := fsutil.WatchFile(*configFile, nil)
		(<-ch).Close() // Drain the first event.
		go func() {
			<-ch
			err := syscall.Exec(os.Args[0], os.Args, os.Environ())
			if err != nil {
				logger.Printf("Unable to Exec:%s: %s\n", os.Args[0], err)
			}
		}()
	}
	httpd.AddHtmlWriter(manager)
	httpd.AddHtmlWriter(logger)
	for _, pathname := range flag.Args() {
		if err := registerSourceDirectory(manager, pathname, "/"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := httpd.StartServer(*portNum, manager, false); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server: %s\n", err)
		os.Exit(1)
	}
}
