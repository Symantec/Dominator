package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
	"log/syslog"
	"os"
)

var (
	fetchInterval = flag.Uint("fetchInterval", 59,
		"Interval between fetches from the MDB source, in seconds")
	hostnameRegex = flag.String("hostnameRegex", ".*",
		"A regular expression to match the desired hostnames")
	mdbFile = flag.String("mdbFile", "/var/lib/Dominator/mdb",
		"Name of file to write filtered MDB data to")
	useSyslog = flag.Bool("syslog", false, "If true, log to syslog")
	url       = flag.String("url", "", "Location of MDB source")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: mdbd [flags...] driver")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Drivers:")
	fmt.Fprintln(os.Stderr,
		"  text: each line contains: host required-image planned-image")
}

type driverFunc func(reader io.Reader, logger *log.Logger) *mdb.Mdb

type driver struct {
	name       string
	driverFunc driverFunc
}

var drivers = []driver{
	{"text", loadText},
}

func getLogger() *log.Logger {
	if *useSyslog {
		logger, err := syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_DAEMON, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return logger
	}
	return log.New(os.Stderr, "mdbd", 0)
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}
	for _, driver := range drivers {
		if flag.Arg(0) == driver.name {
			runDaemon(driver.driverFunc, *url, *mdbFile, *hostnameRegex,
				*fetchInterval, getLogger())
		}
	}
	printUsage()
	os.Exit(2)
}
