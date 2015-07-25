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

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: imagetool [flags...] add|delete|list [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add    name imagefile filterfile")
	fmt.Fprintln(os.Stderr, "  delete name")
	fmt.Fprintln(os.Stderr, "  list")
}

func addImage(name, imageFilename, filterFilename string) error {
	imageFile, err := os.Open(imageFilename)
	if err != nil {
		return err
	}
	defer imageFile.Close()
	filterFile, err := os.Open(filterFilename)
	if err != nil {
		return err
	}
	defer filterFile.Close()
	return nil
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	switch {
	case flag.Arg(0) == "add":
		if flag.NArg() != 4 {
			printUsage()
			os.Exit(2)
		}
		err := addImage(flag.Arg(1), flag.Arg(2), flag.Arg(3))
		if err != nil {
			fmt.Printf("Error adding image\t%s\n", err)
			os.Exit(1)
		}
	case flag.Arg(0) == "delete":
		if flag.NArg() != 2 {
			printUsage()
			os.Exit(2)
		}
	case flag.Arg(0) == "list":
		if flag.NArg() != 1 {
			printUsage()
			os.Exit(2)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}
