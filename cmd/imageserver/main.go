package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"os"
)

var (
	debug   = flag.Bool("debug", false, "If true, show debugging output")
	portNum = flag.Uint("portNum", 6971,
		"Port number to allocate and listen on for HTTP/RPC")
	dataDir = flag.String("stateDir", "/var/lib/imageserver",
		"Name of image server data directory.")
)

func main() {
	flag.Parse()
	if os.Geteuid() == 0 {
		fmt.Println("Do not run the Image Server as root")
		os.Exit(1)
	}
	fi, err := os.Lstat(*dataDir)
	if err != nil {
		fmt.Printf("Cannot stat: %s\t%s\n", *dataDir, err)
		os.Exit(1)
	}
	if !fi.IsDir() {
		fmt.Printf("%s is not a directory\n", *dataDir)
		os.Exit(1)
	}
	imdb, err := scanner.LoadImageDataBase(*dataDir)
	if err != nil {
		fmt.Printf("Cannot load image database\t%s\n", err)
		os.Exit(1)
	}
	fmt.Println(imdb)
}
