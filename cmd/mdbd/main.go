package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/tricorder/go/tricorder"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	datacentre = flag.String("datacentre", "",
		"Datacentre to limit results to (may not be supported by all drivers)")
	debug         = flag.Bool("debug", false, "If true, show debugging output")
	fetchInterval = flag.Uint("fetchInterval", 59,
		"Interval between fetches from the MDB source, in seconds")
	hostnameRegex = flag.String("hostnameRegex", ".*",
		"A regular expression to match the desired hostnames")
	mdbFile = flag.String("mdbFile", "/var/lib/Dominator/mdb",
		"Name of file to write filtered MDB data to")
	portNum = flag.Uint("portNum", constants.SimpleMdbServerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	sourcesFile = flag.String("sourcesFile",
		"/var/lib/Dominator/mdb.sources.list",
		"Name of file list of driver url pairs")
	pidfile = flag.String("pidfile", "", "Name of file to write my PID to")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: mdbd [flags...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Drivers:")
	fmt.Fprintln(os.Stderr,
		"  aws: Amazon AWS endpoint. First arg is datacenter like 'us-east-1'.")
	fmt.Fprintln(os.Stderr,
		"       Second arg is the profile to use out of ~/.aws/credentials which")
	fmt.Fprintln(os.Stderr,
		"       contains the amazon aws credentials. For additional")
	fmt.Fprintln(os.Stderr,
		"       information see:")
	fmt.Fprintln(os.Stderr,
		"       http://docs.aws.amazon.com/sdk-for-go/latest/v1/developerguide/sdkforgo-dg.pdf")
	fmt.Fprintln(os.Stderr,
		"  cis: Cloud Intelligence Service endpoint")
	fmt.Fprintln(os.Stderr,
		"  ds.host.fqdn: JSON with map of map of hosts with fqdn entries")
	fmt.Fprintln(os.Stderr,
		"  text: each line contains: host required-image planned-image")
}

type driver struct {
	name      string
	minArgs   int
	maxArgs   int
	setupFunc makeGeneratorFunc
}

var drivers = []driver{
	{"aws", 2, 2, newAwsGenerator},
	{"cis", 1, 1, newCisGenerator},
	{"ds.host.fqdn", 1, 1, newDsHostFqdnGenerator},
	{"text", 1, 1, newTextGenerator},
}

func gracefulCleanup() {
	os.Remove(*pidfile)
	os.Exit(1)
}

func writePidfile() {
	file, err := os.Create(*pidfile)
	if err != nil {
		return
	}
	defer file.Close()
	fmt.Fprintln(file, os.Getpid())
}

func handleSignals(logger *log.Logger) {
	if *pidfile == "" {
		return
	}
	sigtermChannel := make(chan os.Signal)
	signal.Notify(sigtermChannel, syscall.SIGTERM, syscall.SIGINT)
	writePidfile()
	go func() {
		for {
			select {
			case <-sigtermChannel:
				gracefulCleanup()
			}
		}
	}()
}

func showErrorAndDie(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	tricorder.RegisterFlags()
	circularBuffer := logbuf.New()
	logger := log.New(circularBuffer, "", log.LstdFlags)
	// We have to have inputs.
	if *sourcesFile == "" {
		printUsage()
		os.Exit(2)
	}
	handleSignals(logger)
	readerChannel := fsutil.WatchFile(*sourcesFile, logger)
	file, err := os.Open(*sourcesFile)
	if err != nil {
		showErrorAndDie(err)
	}
	(<-readerChannel).Close()
	generators, err := setupGenerators(file, drivers)
	file.Close()
	if err != nil {
		showErrorAndDie(err)
	}
	httpSrv, err := startHttpServer(*portNum)
	if err != nil {
		showErrorAndDie(err)
	}
	httpSrv.AddHtmlWriter(circularBuffer)
	rpcd := startRpcd(logger)
	go runDaemon(generators, *mdbFile, *hostnameRegex, *datacentre,
		*fetchInterval, func(old, new *mdb.Mdb) {
			rpcd.pushUpdateToAll(old, new)
			httpSrv.UpdateMdb(new)
		},
		logger, *debug)
	<-readerChannel
	fsutil.WatchFileStop()
	if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
		logger.Printf("Unable to Exec:%s: %s\n", os.Args[0], err)
	}
}
