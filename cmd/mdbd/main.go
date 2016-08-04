package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
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
		"Usage: mdbd [flags...] driver0 url0 driver1 url1 ...")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Drivers:")
	fmt.Fprintln(os.Stderr,
		"  cis: Cloud Intelligence Service endpoint")
	fmt.Fprintln(os.Stderr,
		"  ds.host.fqdn: JSON with map of map of hosts with fqdn entries")
	fmt.Fprintln(os.Stderr,
		"  text: each line contains: host required-image planned-image")
	fmt.Fprintln(os.Stderr,
		"  aws: Amazon AWS endpoint. url is datacenter like 'us-east-1'.")
	fmt.Fprintln(os.Stderr,
		"       This driver requires the file ~/.aws/credentials which")
	fmt.Fprintln(os.Stderr,
		"       contains the amazon aws credentials. For additional")
	fmt.Fprintln(os.Stderr,
		"       information see:")
	fmt.Fprintln(os.Stderr,
		"       http://docs.aws.amazon.com/sdk-for-go/latest/v1/developerguide/sdkforgo-dg.pdf")
}

type driverFunc func(reader io.Reader, datacentre string,
	logger *log.Logger) (*mdb.Mdb, error)

type driver struct {
	name       string
	driverFunc driverFunc
}

var drivers = []driver{
	{"cis", loadCis},
	{"ds.host.fqdn", loadDsHostFqdn},
	{"text", loadText},
	// aws driver is handled as a special case for now. See getSource
	// function in this file.
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

// The generator interface generates an mdb from some source.
type generator interface {
	Generate(datacentre string, logger *log.Logger) (*mdb.Mdb, error)
}

// source implements the generator interface and generates an mdb from
// either a flat file or a url.
type source struct {
	// The function parses the data from url or flat file.
	driverFunc driverFunc
	// the url or path of the flat file
	url string
}

func (s source) Generate(
	datacentre string, logger *log.Logger) (*mdb.Mdb, error) {
	return loadMdb(s.driverFunc, s.url, datacentre, logger)
}

func getSource(driverName, url string) generator {
	// special case for aws.
	if driverName == "aws" {
		// With aws, we must know the datacentre up front
		result, err := newAwsGenerator(url)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return result
	}
	for _, driver := range drivers {
		if driverName == driver.name {
			return source{driver.driverFunc, url}
		}
	}
	printUsage()
	os.Exit(2)
	return source{}
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

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg()%2 != 0 {
		printUsage()
		os.Exit(2)
	}
	// We have to have at least one input.
	if *sourcesFile == "" && flag.NArg() == 0 {
		printUsage()
		os.Exit(2)
	}
	logger := getLogger()
	handleSignals(logger)
	generators := make([]generator, 0, flag.NArg()/2)
	if *sourcesFile != "" {
		file, err := os.Open(*sourcesFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) == 2 {
				if fields[0][0] == '#' {
					continue
				}
				generators = append(generators, getSource(fields[0], fields[1]))
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	for index := 0; index < flag.NArg(); index += 2 {
		generators = append(
			generators,
			getSource(
				flag.Arg(index),
				flag.Arg(index+1)))
	}
	runDaemon(
		generators, *mdbFile, *hostnameRegex, *datacentre,
		*fetchInterval, logger, *debug)
}
