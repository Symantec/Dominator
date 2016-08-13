package main

import (
	"encoding/gob"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
	"time"
)

func pollSubcommand(getSubClient getSubClientFunc, args []string) {
	var err error
	var srpcClient *srpc.Client
	for iter := 0; *numPolls < 0 || iter < *numPolls; iter++ {
		if iter > 0 {
			time.Sleep(time.Duration(*interval) * time.Second)
		}
		if srpcClient == nil {
			srpcClient = getSubClient()
		}
		var request sub.PollRequest
		var reply sub.PollResponse
		request.ShortPollOnly = *shortPoll
		pollStartTime := time.Now()
		err = client.CallPoll(srpcClient, request, &reply)
		fmt.Printf("Poll duration: %s\n", time.Since(pollStartTime))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling\t%s\n", err)
			os.Exit(1)
		}
		if *newConnection {
			srpcClient.Close()
			srpcClient = nil
		}
		fs := reply.FileSystem
		if fs == nil {
			if !*shortPoll {
				fmt.Println("No FileSystem pointer")
			}
		} else {
			fs.RebuildInodePointers()
			if *debug {
				fs.List(os.Stdout)
			} else {
				fmt.Println(fs)
			}
			fmt.Printf("Num objects: %d\n", len(reply.ObjectCache))
			if *file != "" {
				f, err := os.Create(*file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating: %s\t%s\n",
						*file, err)
					os.Exit(1)
				}
				encoder := gob.NewEncoder(f)
				encoder.Encode(fs)
				f.Close()
			}
		}
		if reply.LastSuccessfulImageName != "" {
			fmt.Printf("Last successful image: \"%s\"\n",
				reply.LastSuccessfulImageName)
		}
	}
	time.Sleep(time.Duration(*wait) * time.Second)
}
