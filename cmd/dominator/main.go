package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/herd"
	"github.com/Symantec/Dominator/dom/mdb"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/logbuf"
	"log"
	"os"
	"path"
	"runtime"
	"time"
)

var (
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	minInterval = flag.Uint("minInterval", 1,
		"Minimum interval between loops (in seconds)")
	portNum = flag.Uint("portNum", constants.DomPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/Dominator",
		"Name of dominator state directory.")
)

func showMdb(mdb *mdb.Mdb) {
	fmt.Println()
	mdb.DebugWrite(os.Stdout)
	fmt.Println()
}

func main() {
	flag.Parse()
	if os.Geteuid() == 0 {
		fmt.Println("Do not run the Dominator as root")
		os.Exit(1)
	}
	fi, err := os.Lstat(*stateDir)
	if err != nil {
		fmt.Printf("Cannot stat: %s\t%s\n", *stateDir, err)
		os.Exit(1)
	}
	if !fi.IsDir() {
		fmt.Printf("%s is not a directory\n", *stateDir)
		os.Exit(1)
	}
	interval := time.Duration(*minInterval) * time.Second
	circularBuffer := logbuf.New(*logbufLines)
	logger := log.New(circularBuffer, "", log.LstdFlags)
	mdbChannel := mdb.StartMdbDaemon(path.Join(*stateDir, "mdb"), logger)
	herd := herd.NewHerd(fmt.Sprintf("%s:%d", *imageServerHostname,
		*imageServerPortNum), logger)
	herd.AddHtmlWriter(circularBuffer)
	err = herd.StartServer(*portNum, true)
	if err != nil {
		fmt.Printf("Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
	nextCycleStopTime := time.Now().Add(interval)
	for {
		select {
		case mdb := <-mdbChannel:
			herd.MdbUpdate(mdb)
			if *debug {
				showMdb(mdb)
			}
			runtime.GC() // An opportune time to take out the garbage.
		default:
			// Do work.
			if herd.PollNextSub() {
				if *debug {
					fmt.Print(".")
				}
				time.Sleep(nextCycleStopTime.Sub(time.Now()))
				nextCycleStopTime = time.Now().Add(interval)
				runtime.GC() // An opportune time to take out the garbage.
			}
		}
	}
}
