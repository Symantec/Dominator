package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/fleet"
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
	stateDir = flag.String("stateDir", "/tmp/Dominator",
		"Name of dominator state directory.")
)

func main() {
	flag.Parse()
	mdbChannel := mdb.StartMdbDaemon(path.Join(*stateDir, "mdb"))
	interval, _ := time.ParseDuration(fmt.Sprintf("%ds", *minInterval))
	var fleet fleet.Fleet
	for {
		minCycleStopTime := time.Now().Add(interval)
		select {
		case mdb := <-mdbChannel:
			fleet.MdbUpdate(mdb)
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
			fleet.ScanNextSub()
		}
		fmt.Print(".")
		runtime.GC() // An opportune time to take out the garbage.
		time.Sleep(minCycleStopTime.Sub(time.Now()))
	}
}
