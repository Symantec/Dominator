package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/herd"
	"github.com/Symantec/Dominator/dom/mdb"
	"os"
	"path"
	"runtime"
	"time"
)

var (
	debug       = flag.Bool("debug", false, "If true, show debugging output")
	minInterval = flag.Uint("minInterval", 1,
		"Minimum interval between loops (in seconds)")
	portNum = flag.Uint("portNum", 6970,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/Dominator",
		"Name of dominator state directory.")
)

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
	mdbChannel := mdb.StartMdbDaemon(path.Join(*stateDir, "mdb"))
	interval, _ := time.ParseDuration(fmt.Sprintf("%ds", *minInterval))
	var herd herd.Herd
	for {
		minCycleStopTime := time.Now().Add(interval)
		select {
		case mdb := <-mdbChannel:
			herd.MdbUpdate(mdb)
			if *debug {
				b, _ := json.Marshal(mdb)
				var out bytes.Buffer
				json.Indent(&out, b, "", "    ")
				fmt.Println()
				out.WriteTo(os.Stdout)
				fmt.Println()
			}
		default:
			// Do work.
			herd.PollNextSub()
		}
		fmt.Print(".")
		runtime.GC() // An opportune time to take out the garbage.
		time.Sleep(minCycleStopTime.Sub(time.Now()))
	}
}
