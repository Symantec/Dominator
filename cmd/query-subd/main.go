package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
	"os"
	"strings"
	"time"
)

var (
	file     = flag.String("file", "", "Name of file to write encoded data to")
	debug    = flag.Bool("debug", false, "Enable debug mode")
	interval = flag.Uint("interval", 1, "Seconds to sleep between Polls")
	numPolls = flag.Int("numPolls", 1,
		"The number of polls to run (infinite: < 0)")
	newConnection = flag.Bool("newConnection", false,
		"If true, (re)open a connection for each Poll")
)

func main() {
	flag.Parse()
	args := flag.Args()
	clientName := args[0]
	if !strings.Contains(clientName, ":") {
		clientName = clientName + ":6969"
	}
	var client *rpc.Client
	var err error
	sleepDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", *interval))
	for iter := 0; *numPolls < 0 || iter < *numPolls; iter++ {
		if iter > 0 {
			time.Sleep(sleepDuration)
		}
		if client == nil {
			client, err = rpc.DialHTTP("tcp", clientName)
			if err != nil {
				fmt.Printf("Error dialing\t%s\n", err)
				os.Exit(1)
			}
		}
		arg := new(uint64)
		var reply *scanner.FileSystem
		err = client.Call("Subd.Poll", arg, &reply)
		if err != nil {
			fmt.Printf("Error calling\t%s\n", err)
			os.Exit(1)
		}
		if *newConnection {
			client.Close()
			client = nil
		}
		reply.RebuildPointers()
		if *debug {
			reply.DebugWrite(os.Stdout, "")
		} else {
			fmt.Print(reply)
		}
		if *file != "" {
			f, err := os.Create(*file)
			if err != nil {
				fmt.Printf("Error creating: %s\t%s\n", *file, err)
				os.Exit(1)
			}
			encoder := gob.NewEncoder(f)
			encoder.Encode(reply)
			f.Close()
		}
	}
}
