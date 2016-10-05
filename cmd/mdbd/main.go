package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/tricorder/go/tricorder"
	"log"
	"os"
	"os/signal"
	"strings"
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
	name       string
	driverFunc driverFunc
}

var drivers = []driver{
	// aws driver is handled as a special case for now. See getSource
	// function in this file.
	{"cis", loadCis},
	{"ds.host.fqdn", loadDsHostFqdn},
	{"text", loadText},
}

func getSource(driverAndArgs []string) (
	result generator, err error) {
	if len(driverAndArgs) == 0 {
		return nil, errors.New("At least driver name expected.")
	}
	// Special case for aws.
	if driverAndArgs[0] == "aws" {
		if len(driverAndArgs) != 3 {
			return nil, errors.New("aws expects 2 args: datacenter and profile.")
		}
		// [1] is the datacenter and [2] is the profile
		result, err := newAwsGenerator(driverAndArgs[1], driverAndArgs[2])
		if err != nil {
			showErrorAndDie(err)
		}
		return result, nil
	}
	if len(driverAndArgs) != 2 {
		return nil, errors.New("1 arg expected: url.")
	}
	for _, driver := range drivers {
		if driverAndArgs[0] == driver.name {
			return source{driver.driverFunc, driverAndArgs[1]}, nil
		}
	}
	printUsage()
	os.Exit(2)
	return
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
	var generators []generator
	readerChannel := fsutil.WatchFile(*sourcesFile, logger)
	file, err := os.Open(*sourcesFile)
	if err != nil {
		showErrorAndDie(err)
	}
	(<-readerChannel).Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || len(fields[0]) == 0 || fields[0][0] == '#' {
			continue
		}
		gen, err := getSource(fields)
		if err != nil {
			showErrorAndDie(err)
		}
		generators = append(generators, gen)
	}
	if err := scanner.Err(); err != nil {
		showErrorAndDie(err)
	}
	file.Close()
	httpSrv, err := startHttpServer(*portNum)
	if err != nil {
		showErrorAndDie(err)
	}
	httpSrv.AddHtmlWriter(circularBuffer)
	updateFunc := startRpcd(logger)
	go runDaemon(generators, *mdbFile, *hostnameRegex, *datacentre,
		*fetchInterval, updateFunc, logger, *debug)
	<-readerChannel
	fsutil.WatchFileStop()
	if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
		logger.Printf("Unable to Exec:%s: %s\n", os.Args[0], err)
	}
}
