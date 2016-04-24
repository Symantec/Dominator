package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/Symantec/Dominator/lib/x509util"
	"io/ioutil"
	"os"
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: show-cert certfile")
}

func main() {
	if len(os.Args) != 2 {
		printUsage()
		os.Exit(2)
	}
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read certfile: %s\n", err)
		os.Exit(1)
	}
	block, rest := pem.Decode(data)
	if block == nil {
		fmt.Fprintf(os.Stderr, "Failed to parse certificate PEM")
		os.Exit(1)
	}
	if len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "%d extra bytes in certfile\n", len(rest))
		os.Exit(1)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse certificate: %s\n", err)
		os.Exit(1)
	}
	permittedMethods, err := x509util.GetPermittedMethods(cert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get methods: %s\n", err)
		os.Exit(1)
	}
	if len(permittedMethods) > 0 {
		fmt.Println("Permitted methods:")
		for method := range permittedMethods {
			fmt.Println(" ", method)
		}
	} else {
		fmt.Println("No methods are permitted")
	}
}
