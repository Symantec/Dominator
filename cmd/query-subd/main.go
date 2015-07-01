package main

import (
	"encoding/gob"
	"fmt"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
	"os"
)

func main() {
	client, err := rpc.DialHTTP("tcp", os.Args[1]+":6969")
	if err != nil {
		fmt.Printf("Error dialing\t%s\n", err)
		os.Exit(1)
	}
	arg := new(uint64)
	var reply *scanner.FileSystem
	err = client.Call("Subd.Poll", arg, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		os.Exit(1)
	}
	fmt.Print(reply)
	if len(os.Args) >= 3 {
		file, err := os.Create(os.Args[2])
		if err != nil {
			fmt.Printf("Error creating: %s\t%s\n", os.Args[2], err)
			os.Exit(1)
		}
		encoder := gob.NewEncoder(file)
		encoder.Encode(reply)
		file.Close()
	}
	if len(os.Args) >= 4 {
		file, err := os.Create(os.Args[3])
		if err != nil {
			fmt.Printf("Error creating: %s\t%s\n", os.Args[3], err)
			os.Exit(1)
		}
		encoder := gob.NewEncoder(file)
		encoder.Encode(reply)
		file.Close()
	}
}
