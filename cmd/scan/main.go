package main

// Benchmark the scanning of a file-system tree.

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"syscall"
	"time"
)

var (
	interval = flag.Uint("interval", 0, "Seconds to sleep after each scan")
	numScans = flag.Int("numScans", 1,
		"The number of scans to run (infinite: < 0)")
	rootDir = flag.String("rootDir", "/",
		"Name of root of directory tree to scan")
)

func main() {
	flag.Parse()
	var err error
	bytesPerSecond, blocksPerSecond, err := fsbench.GetReadSpeed(*rootDir)
	if err != nil {
		fmt.Printf("Error! %s\n", err)
		return
	}
	ctx := fsrateio.NewContext(bytesPerSecond, blocksPerSecond)
	fmt.Println(ctx)
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
	var prev_fs *scanner.FileSystem
	sleepDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", *interval))
	for iter := 1; *numScans < 0 || iter <= *numScans; iter++ {
		timeStart := time.Now()
		fs, err := scanner.ScanFileSystem(*rootDir, "", ctx)
		timeStop := time.Now()
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			return
		}
		fmt.Print(fs)
		var tread uint64 = 0
		for _, inode := range fs.InodeTable {
			tread += inode.Size
		}
		fmt.Printf("Total scanned: %s,\t", fsrateio.FormatBytes(tread))
		bytesPerSecond := uint64(float64(tread) /
			timeStop.Sub(timeStart).Seconds())
		fmt.Printf("%s/s\n", fsrateio.FormatBytes(bytesPerSecond))
		if prev_fs != nil {
			if !scanner.Compare(prev_fs, fs, os.Stdout) {
				fmt.Println("Scan results different from last run")
			}
		}
		prev_fs = fs
		time.Sleep(sleepDuration)
	}
}
