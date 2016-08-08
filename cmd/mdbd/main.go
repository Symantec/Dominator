package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/syslog"
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
	sourcesFile = flag.String("sourcesFile", "",
		"Name of file list of driver url pairs")
	useSyslog = flag.Bool("syslog", false, "If true, log to syslog")
	pidfile   = flag.String("pidfile", "", "Name of file to write my PID to")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: mdbd [flags...] driver0 url0 driver1 url1 addparam1 driver2 driver3 url3...")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Drivers:")
	fmt.Fprintln(os.Stderr,
		"  aws: Amazon AWS endpoint. first arg is datacenter like 'us-east-1'.")
	fmt.Fprintln(os.Stderr,
		"       second arg is the profile to use out of ~/.aws/credentials which")
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

func getLogger() *log.Logger {
	if *useSyslog {
		s, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "mdbd")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return log.New(s, "", 0)
	}
	return log.New(os.Stderr, "", log.LstdFlags)
}

func getSource(driverAndArgs []string) (
	result generator, argsTaken int, err error) {
	if len(driverAndArgs) == 0 {
		return nil, 0, errors.New("At least driver name expected.")
	}
	// Special case for aws.
	if driverAndArgs[0] == "aws" {
		if len(driverAndArgs) < 3 {
			return nil, 0, errors.New("aws expects 2 args: datacenter and profile.")
		}
		// [1] is the datacenter and [2] is the profile
		result, err := newAwsGenerator(driverAndArgs[1], driverAndArgs[2])
		if err != nil {
			showErrorAndDie(err)
		}
		return result, 3, nil
	}
	if len(driverAndArgs) < 2 {
		return nil, 0, errors.New("1 arg expected: url.")
	}
	for _, driver := range drivers {
		if driverAndArgs[0] == driver.name {
			return source{driver.driverFunc, driverAndArgs[1]}, 2, nil
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
	// We have to have at least one input.
	if *sourcesFile == "" && flag.NArg() == 0 {
		printUsage()
		os.Exit(2)
	}
	logger := getLogger()
	handleSignals(logger)
	var generators []generator
	if *sourcesFile != "" {
		file, err := os.Open(*sourcesFile)
		if err != nil {
			showErrorAndDie(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) == 0 || len(fields[0]) == 0 || fields[0][0] == '#' {
				continue
			}
			gen, argsTaken, err := getSource(fields)
			if err != nil {
				showErrorAndDie(err)
			}
			if argsTaken != len(fields) {
				showErrorAndDie(
					errors.New(
						fmt.Sprintf(
							"Too many args provided: %v",
							fields)))
			}
			generators = append(generators, gen)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	flagArguments := make([]string, flag.NArg())
	for index := 0; index < flag.NArg(); index++ {
		flagArguments[index] = flag.Arg(index)
	}
	for index := 0; index < len(flagArguments); {
		gen, argsTaken, err := getSource(flagArguments[index:])
		if err != nil {
			showErrorAndDie(err)
		}
		generators = append(generators, gen)
		index += argsTaken
	}
	runDaemon(generators, *mdbFile, *hostnameRegex, *datacentre, *fetchInterval,
		logger, *debug)
}
