package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/x509util"
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
	now := time.Now()
	if notYet := cert.NotBefore.Sub(now); notYet > 0 {
		fmt.Fprintf(os.Stderr, "  Will not be valid for %s\n",
			format.Duration(notYet))
	}
	if expired := now.Sub(cert.NotAfter); expired > 0 {
		fmt.Fprintf(os.Stderr, "  Expired %s ago\n", format.Duration(expired))
	}
	username, err := x509util.GetUsername(cert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get username: %s\n", err)
		return
	}
	fmt.Printf("  Issued to: %s\n", username)
	permittedMethods, err := x509util.GetPermittedMethods(cert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get methods: %s\n", err)
		return
	}
	if len(permittedMethods) > 0 {
		fmt.Println("  Permitted methods:")
		showList(permittedMethods)
	} else {
		fmt.Println("  No methods are permitted")
	}
	groupList, err := x509util.GetGroupList(cert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get group list: %s\n", err)
		return
	}
	if len(groupList) > 0 {
		fmt.Println("  Group list:")
		showList(groupList)
	} else {
		fmt.Println("  No group memberships")
	}
}

func showList(list map[string]struct{}) {
	sortedList := make([]string, 0, len(list))
	for entry := range list {
		sortedList = append(sortedList, entry)
	}
	sort.Strings(sortedList)
	for _, entry := range sortedList {
		fmt.Println("   ", entry)
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
