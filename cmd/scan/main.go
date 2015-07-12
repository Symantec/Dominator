package main

// Benchmark the scanning of a file-system tree.

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/lib/memstats"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"runtime"
	"syscall"
	"time"
)

var (
	debugFile = flag.String("debugFile", "",
		"Name of file to write debugging information to")
	interval = flag.Uint("interval", 0, "Seconds to sleep after each scan")
	numScans = flag.Int("numScans", 1,
		"The number of scans to run (infinite: < 0)")
	objectCache = flag.String("objectCache", "",
		"Name of directory containing the object cache")
	rootDir = flag.String("rootDir", "/",
		"Name of root of directory tree to scan")
	rpcFile = flag.String("rpcFile", "",
		"Name of file to write encoded data to")
	scanSpeed = flag.Uint("scanSpeed", 0,
		"Scan speed in percent of maximum (0: default)")
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
	if *scanSpeed != 0 {
		ctx.SetSpeedPercent(*scanSpeed)
	}
	fmt.Println(ctx)
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
	var prev_fs *scanner.FileSystem
	sleepDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", *interval))
	for iter := 1; *numScans < 0 || iter <= *numScans; iter++ {
		timeStart := time.Now()
		fs, err := scanner.ScanFileSystem(*rootDir, *objectCache, ctx)
		timeStop := time.Now()
		if iter > 1 {
			fmt.Println()
		}
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			return
		}
		fmt.Print(fs)
		fmt.Printf("Total scanned: %s,\t",
			fsrateio.FormatBytes(fs.TotalDataBytes))
		bytesPerSecond := uint64(float64(fs.TotalDataBytes) /
			timeStop.Sub(timeStart).Seconds())
		fmt.Printf("%s/s\n", fsrateio.FormatBytes(bytesPerSecond))
		if prev_fs != nil {
			if !scanner.Compare(prev_fs, fs, os.Stdout) {
				fmt.Println("Scan results different from last run")
			}
		}
		runtime.GC() // Clean up before showing memory statistics.
		memstats.WriteMemoryStats(os.Stdout)
		if *debugFile != "" {
			file, err := os.Create(*debugFile)
			if err != nil {
				fmt.Printf("Error! %s\n", err)
				return
			}
			w = bufio.NewWriter(file)
			fs.DebugWrite(w, "")
			w.Flush()
			file.Close()
		}
		if *rpcFile != "" {
			file, err := os.Create(*rpcFile)
			if err != nil {
				fmt.Printf("Error creating: %s\t%s\n", *rpcFile, err)
				os.Exit(1)
			}
			encoder := gob.NewEncoder(file)
			encoder.Encode(fs)
			file.Close()
		}
		prev_fs = fs
		time.Sleep(sleepDuration)
	}
}
