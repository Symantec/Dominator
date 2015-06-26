package main

// Benchmark the scanning of a file-system tree.
// Usage: scan [dirname [numScans]]
//   dirname:   the top of the directory tree to scan (default=/)
//   numScans:  the number of scans to run (default=1, infinite: < 0)

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"strconv"
	"syscall"
	"time"
)

func main() {
	pathname := "/"
	var numScans int = 1
	var err error
	if len(os.Args) >= 2 {
		pathname = os.Args[1]
	}
	if len(os.Args) == 3 {
		numScans, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			return
		}
	}
	ctx, err := fsrateio.NewContext(pathname)
	if err != nil {
		fmt.Printf("Error! %s\n", err)
		return
	}
	fmt.Println(ctx)
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
	var prev_fs *scanner.FileSystem
	for iter := 1; numScans < 0 || iter <= numScans; iter++ {
		timeStart := time.Now()
		fs, err := scanner.ScanFileSystem(pathname, ctx)
		timeStop := time.Now()
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			return
		}
		var tread uint64 = 0
		for _, inode := range fs.InodeTable {
			tread += inode.Length()
		}
		fmt.Printf("Total scanned: %s,\t", fsrateio.FormatBytes(tread))
		bytesPerSecond := uint64(float64(tread) /
			timeStop.Sub(timeStart).Seconds())
		fmt.Printf("%s/s\n", fsrateio.FormatBytes(bytesPerSecond))
		if prev_fs != nil {
			if !scanner.Compare(prev_fs, fs, nil) {
				fmt.Println("Scan results different from last run")
			}
		}
		prev_fs = fs
	}
}
