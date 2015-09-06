package main

import (
	"encoding/gob"
	"fmt"
	"github.com/Symantec/Dominator/proto/sub"
	"net/rpc"
	"os"
	"time"
)

func pollSubcommand(client *rpc.Client, args []string) {
	var err error
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	for iter := 0; *numPolls < 0 || iter < *numPolls; iter++ {
		if iter > 0 {
			time.Sleep(time.Duration(*interval) * time.Second)
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
		pollStartTime := time.Now()
		err = client.Call("Subd.Poll", request, &reply)
		fmt.Printf("Poll duration: %s\n", time.Since(pollStartTime))
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
			fs.RebuildInodePointers()
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
	time.Sleep(time.Duration(*wait) * time.Second)
}
