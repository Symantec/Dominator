package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/Symantec/Dominator/lib/x509util"
	"io/ioutil"
	"os"
	"sort"
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: show-cert certfile")
}

func showCert(filename string) {
	fmt.Println("Certificate:", filename+":")
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read certfile: %s\n", err)
		return
	}
	block, rest := pem.Decode(data)
	if block == nil {
		fmt.Fprintf(os.Stderr, "Failed to parse certificate PEM")
		return
	}
	if len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "%d extra bytes in certfile\n", len(rest))
		return
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse certificate: %s\n", err)
		return
	}
	permittedMethods, err := x509util.GetPermittedMethods(cert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get methods: %s\n", err)
		return
	}
	if len(permittedMethods) > 0 {
		fmt.Println("  Permitted methods:")
		sortedList := make([]string, 0, len(permittedMethods))
		for method := range permittedMethods {
			sortedList = append(sortedList, method)
		}
		sort.Strings(sortedList)
		for _, method := range sortedList {
			fmt.Println("   ", method)
		}
	} else {
		fmt.Println("  No methods are permitted")
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	for _, filename := range os.Args[1:] {
		showCert(filename)
	}
}
