package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/mdb/mdbd"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
)

var (
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	mdbFile = flag.String("mdbFile", "/var/lib/Dominator/mdb",
		"File to read MDB data from (default format is JSON)")

	numMachines int
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: mdb-relayd [flags...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
}

func showMdb(mdb *mdb.Mdb) {
	mdb.DebugWrite(os.Stdout)
	fmt.Println()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 0 {
		printUsage()
		os.Exit(2)
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	mdbChannel := mdbd.StartMdbDaemon(*mdbFile, logger)
	oldMachines := make(map[string]mdb.Machine)
	firstTime := true
	for mdb := range mdbChannel {
		if *debug {
			showMdb(mdb)
		}
		numNew := 0
		numChanged := 0
		machinesToDelete := make(map[string]struct{})
		for name := range oldMachines {
			machinesToDelete[name] = struct{}{}
		}
		for _, machine := range mdb.Machines {
			if machine.Hostname == "" {
				logger.Println("Received machine with empty Hostname")
				continue
			}
			delete(machinesToDelete, machine.Hostname)
			if oldMachine, ok := oldMachines[machine.Hostname]; ok {
				if !reflect.DeepEqual(oldMachine, machine) {
					oldMachines[machine.Hostname] = machine
					numChanged++
				}
			} else {
				oldMachines[machine.Hostname] = machine
				numNew++
			}
		}
		if firstTime {
			firstTime = false
			logger.Printf("Initial MDB data: %d machines\n",
				len(mdb.Machines))
			continue
		}
		for name := range machinesToDelete {
			delete(oldMachines, name)
		}
		logger.Printf("MDB update: %d new, %d removed, %d changed\n",
			numNew, len(machinesToDelete), numChanged)
	}
}
