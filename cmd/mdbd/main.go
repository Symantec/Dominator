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
	"strings"
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

type source struct {
	driverFunc driverFunc
	url        string
}

func getSource(driverName, url string) source {
	for _, driver := range drivers {
		if driverName == driver.name {
			return source{driver.driverFunc, url}
		}
	}
	printUsage()
	os.Exit(2)
	return source{}
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg()%2 != 0 {
		printUsage()
		os.Exit(2)
	}
	sources := make([]source, 0, flag.NArg()/2)
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
				sources = append(sources, getSource(fields[0], fields[1]))
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	for index := 0; index < flag.NArg(); index += 2 {
		sources = append(sources, getSource(flag.Arg(index), flag.Arg(index+1)))
	}
	runDaemon(sources, *mdbFile, *hostnameRegex, *datacentre, *fetchInterval,
		getLogger(), *debug)
}
