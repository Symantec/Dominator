package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/mdb"
	"os"
	"path"
)

var (
	debug   = flag.Bool("debug", false, "If true, show debugging output")
	portNum = flag.Uint("portNum", 6970,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/tmp/Dominator",
		"Name of dominator state directory.")
)

func main() {
	flag.Parse()
	mdbFileName := path.Join(*stateDir, "mdb")
	mdbChannel := mdb.StartMdbDaemon(mdbFileName)
	for {
		mdb := <-mdbChannel
		if *debug {
			b, _ := json.Marshal(mdb)
			var out bytes.Buffer
			json.Indent(&out, b, "", "    ")
			out.WriteTo(os.Stdout)
			fmt.Println()
		}
	}
}
