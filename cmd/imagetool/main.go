package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"os"
)

var (
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("Missing command")
		os.Exit(2)
	}
	switch {
	case flag.Arg(0) == "add":
		if flag.NArg() != 4 {
			fmt.Println(
				"Usage: imagetool [flags...] add name imagefile filterfile")
			os.Exit(2)
		}
	case flag.Arg(0) == "delete":
		if flag.NArg() != 2 {
			fmt.Println("Usage: imagetool [flags...] delete imagename")
			os.Exit(2)
		}
	case flag.Arg(0) == "list":
		if flag.NArg() != 1 {
			fmt.Println("Usage: imagetool [flags...] list")
			os.Exit(2)
		}
	default:
		fmt.Println("Usage: imagetool [flags...] add|delete|list")
		os.Exit(2)
	}
}
