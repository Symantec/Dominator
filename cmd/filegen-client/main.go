package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/mdbd"
	"github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io"
	"log"
	"os"
	"path"
)

var (
	certFile = flag.String("certFile",
		path.Join(os.Getenv("HOME"), ".ssl/cert.pem"),
		"Name of file containing the user SSL certificate")
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	keyFile = flag.String("keyFile",
		path.Join(os.Getenv("HOME"), ".ssl/key.pem"),
		"Name of file containing the user SSL key")
	mdbFile = flag.String("mdbFile", "/var/lib/Dominator/mdb",
		"File to read MDB data from (default format is JSON)")

	outputSemaphore = make(chan struct{}, 1)
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: filegen-client [flags...] pathname source")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
}

func showMdb(mdb *mdb.Mdb) {
	mdb.DebugWrite(os.Stdout)
	fmt.Println()
}

type machineType struct {
	machine       client.Machine
	updateChannel <-chan []proto.FileInfo
}

func (m *machineType) handleUpdates(objSrv *memory.ObjectServer) {
	for fileInfos := range m.updateChannel {
		outputSemaphore <- struct{}{}
		fmt.Printf("For machine: %s:\n", m.machine.Machine.Hostname)
		for _, fileInfo := range fileInfos {
			fmt.Printf("  pathname: %s\n    hash=%x\n    contents:\n",
				fileInfo.Pathname, fileInfo.Hash)
			if _, reader, err := objSrv.GetObject(fileInfo.Hash); err != nil {
				fmt.Println(err)
			} else {
				io.Copy(os.Stderr, reader)
				fmt.Println("\n-----------------------------------------------")
			}
		}
		<-outputSemaphore
	}
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 2 {
		printUsage()
		os.Exit(2)
	}
	setupTls(*certFile, *keyFile)
	objectServer := memory.NewObjectServer()
	logger := log.New(os.Stdout, "", log.LstdFlags)
	manager := client.New(objectServer, logger)
	mdbChannel := mdbd.StartMdbDaemon(*mdbFile, logger)
	machines := make(map[string]*machineType)
	computedFiles := make([]client.ComputedFile, 1)
	computedFiles[0].Pathname = flag.Arg(0)
	computedFiles[0].Source = flag.Arg(1)
	for {
		select {
		case mdb := <-mdbChannel:
			if *debug {
				showMdb(mdb)
			}
			machinesToDelete := make(map[string]struct{}, len(machines))
			for hostname := range machines {
				machinesToDelete[hostname] = struct{}{}
			}
			for _, mdbEntry := range mdb.Machines {
				delete(machinesToDelete, mdbEntry.Hostname)
				machine := &machineType{
					machine: client.Machine{mdbEntry, computedFiles}}
				if oldMachine, ok := machines[mdbEntry.Hostname]; !ok {
					machine.updateChannel = manager.Add(machine.machine, 1)
					go machine.handleUpdates(objectServer)
				} else {
					oldMachine.machine = machine.machine
					manager.Update(client.Machine{mdbEntry, computedFiles})
				}
			}
			for hostname := range machinesToDelete {
				manager.Remove(hostname)
			}
		}
	}
}
