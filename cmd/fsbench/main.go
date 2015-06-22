package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"os"
)

// Benchmark the read speed of the underlying block device for a given file.
func main() {
	pathname := "/"
	if len(os.Args) == 2 {
		pathname = os.Args[1]
	}
	bytesPerSecond, blocksPerSecond, err := fsbench.GetReadSpeed(pathname)
	if err != nil {
		fmt.Printf("Error! %s\n", err)
		return
	}
	fmt.Printf("speed=%d MiB/s ", bytesPerSecond>>20)
	if blocksPerSecond > 0 {
		fmt.Printf("%d blocks/s\n", blocksPerSecond)
	} else {
		fmt.Println("I/O accounting not available")
	}
}
