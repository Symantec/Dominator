// +build !linux

package main

import (
	"os"
)

func main() {
	os.Stderr.Write([]byte("Not available on this OS\n"))
	os.Exit(1)
}
