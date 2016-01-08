package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	daemon        = flag.Bool("daemon", false, "If true, run in daemon mode")
	fetchInterval = flag.Uint("fetchInterval", 59,
		"Interval between fetches from the MDB source, in seconds")
	mdbFile = flag.String("mdbFile", "/var/lib/Dominator/mdb",
		"Name of file to write filtered MDB data to")
	url  = flag.String("url", "", "Location of MDB source")
	zone = flag.String("zone", "",
		"The zone (typically a datacentre) to select")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: mdbd [flags...] driver")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Drivers:")
}

type driverFunc func(url string)

type driver struct {
	name       string
	driverFunc driverFunc
}

func (driver driver) Run() {
	driver.driverFunc(*url)
	os.Exit(3)
}

var drivers = []driver{}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}
	for _, driver := range drivers {
		if flag.Arg(0) == driver.name {
			driver.Run()
		}
	}
	printUsage()
	os.Exit(2)
}
