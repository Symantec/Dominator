package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
	"os"
	"strings"
)

var (
	file = flag.String("file", "", "Name of file to write encoded data to")
)

func main() {
	flag.Parse()
	args := flag.Args()
	clientName := args[0]
	if !strings.Contains(clientName, ":") {
		clientName = clientName + ":6969"
	}
	client, err := rpc.DialHTTP("tcp", clientName)
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
