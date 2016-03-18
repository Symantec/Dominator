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
	"time"
)

var (
	benchmark = flag.Bool("benchmark", false,
		"If true, perform benchmark timing")
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

	numMachines int
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

type messageType struct {
	hostname  string
	fileInfos []proto.FileInfo
}

func benchmarkMessageHandler(messageChannel <-chan messageType) {
	numMessages := 0
	startTime := time.Now()
	for message := range messageChannel {
		numMessages++
		if numMessages == numMachines {
			fmt.Printf("Time taken: %s\n", time.Since(startTime))
		} else if numMessages > numMachines {
			fmt.Printf("Extra message for machine: %s\n", message.hostname)
		}
	}
}

func displayMessageHandler(messageChannel <-chan messageType,
	objSrv *memory.ObjectServer) {
	for message := range messageChannel {
		fmt.Printf("For machine: %s:\n", message.hostname)
		for _, fileInfo := range message.fileInfos {
			fmt.Printf("  pathname: %s\n    hash=%x\n    contents:\n",
				fileInfo.Pathname, fileInfo.Hash)
			if _, reader, err := objSrv.GetObject(fileInfo.Hash); err != nil {
				fmt.Println(err)
			} else {
				io.Copy(os.Stdout, reader)
				fmt.Println("\n-----------------------------------------------")
			}
		}
	}
}

func handleUpdates(hostname string, updateChannel <-chan []proto.FileInfo,
	messageChannel chan<- messageType) {
	for fileInfos := range updateChannel {
		messageChannel <- messageType{hostname, fileInfos}
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
	machines := make(map[string]struct{})
	computedFiles := make([]client.ComputedFile, 1)
	computedFiles[0].Pathname = flag.Arg(0)
	computedFiles[0].Source = flag.Arg(1)
	messageChannel := make(chan messageType, 1)
	if *benchmark {
		go benchmarkMessageHandler(messageChannel)
	} else {
		go displayMessageHandler(messageChannel, objectServer)
	}
	for {
		select {
		case mdb := <-mdbChannel:
			if *debug {
				showMdb(mdb)
			}
			numMachines = len(mdb.Machines)
			machinesToDelete := make(map[string]struct{}, len(machines))
			for hostname := range machines {
				machinesToDelete[hostname] = struct{}{}
			}
			for _, mdbEntry := range mdb.Machines {
				delete(machinesToDelete, mdbEntry.Hostname)
				machine := client.Machine{mdbEntry, computedFiles}
				if _, ok := machines[mdbEntry.Hostname]; !ok {
					machines[mdbEntry.Hostname] = struct{}{}
					go handleUpdates(mdbEntry.Hostname, manager.Add(machine, 1),
						messageChannel)
				} else {
					manager.Update(client.Machine{mdbEntry, computedFiles})
				}
			}
			for hostname := range machinesToDelete {
				manager.Remove(hostname)
				delete(machines, hostname)
			}
		}
	}
}
