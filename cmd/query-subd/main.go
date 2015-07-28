package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/proto/sub"
	"net/rpc"
	"os"
	"time"
)

var (
	debug         = flag.Bool("debug", false, "Enable debug mode")
	file          = flag.String("file", "", "Name of file to write encoded data to")
	interval      = flag.Uint("interval", 1, "Seconds to sleep between Polls")
	newConnection = flag.Bool("newConnection", false,
		"If true, (re)open a connection for each Poll")
	numPolls = flag.Int("numPolls", 1,
		"The number of polls to run (infinite: < 0)")
	subPortNum = flag.Uint("subPortNum", constants.SubPortNumber,
		"Port number of sub")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: query-subd [flags...] hostname")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}
	args := flag.Args()
	clientName := fmt.Sprintf("%s:%d", args[0], *subPortNum)
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
		var request sub.PollRequest
		var reply sub.PollResponse
		err = client.Call("Subd.Poll", request, &reply)
		if err != nil {
			fmt.Printf("Error calling\t%s\n", err)
			os.Exit(1)
		}
		if *newConnection {
			client.Close()
			client = nil
		}
		fs := reply.FileSystem
		if fs == nil {
			fmt.Println("No FileSystem pointer")
		} else {
			fs.RebuildPointers()
			if *debug {
				fs.DebugWrite(os.Stdout, "")
			} else {
				fmt.Print(fs)
			}
			if *file != "" {
				f, err := os.Create(*file)
				if err != nil {
					fmt.Printf("Error creating: %s\t%s\n", *file, err)
					os.Exit(1)
				}
				encoder := gob.NewEncoder(f)
				encoder.Encode(fs)
				f.Close()
			}
		}
	}
}
