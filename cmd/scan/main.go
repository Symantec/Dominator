package main

// Scan a file-system similar to subd.

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"github.com/Symantec/Dominator/lib/memstats"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"runtime"
	"syscall"
	"time"
)

var (
	debugFile = flag.String("debugFile", "",
		"Name of file to write debugging information to")
	gobFile = flag.String("gobFile", "",
		"Name of file to write encoded data to")
	interval = flag.Uint("interval", 0, "Seconds to sleep after each scan")
	numScans = flag.Int("numScans", 1,
		"The number of scans to run (infinite: < 0)")
	objectCache = flag.String("objectCache", "",
		"Name of directory containing the object cache")
	rootDir = flag.String("rootDir", "/",
		"Name of root of directory tree to scan")
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
	var configuration scanner.Configuration
	configuration.ScanFilter, err = filter.NewFilter(nil)
	if err != nil {
		fmt.Printf("Unable to create empty filter\t%s\n", err)
		os.Exit(1)
	}
	configuration.FsScanContext = fsrateio.NewReaderContext(bytesPerSecond,
		blocksPerSecond, 0)
	if *scanSpeed != 0 {
		configuration.FsScanContext.GetContext().SetSpeedPercent(*scanSpeed)
	}
	fmt.Println(configuration.FsScanContext)
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
	var prev_fs *scanner.FileSystem
	for iter := 0; *numScans < 0 || iter < *numScans; iter++ {
		timeStart := time.Now()
		fs, err := scanner.ScanFileSystem(*rootDir, *objectCache,
			&configuration)
		timeStop := time.Now()
		if iter > 0 {
			fmt.Println()
		}
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			return
		}
		fmt.Print(fs)
		fmt.Printf("Total scanned: %s,\t",
			format.FormatBytes(fs.TotalDataBytes))
		bytesPerSecond := uint64(float64(fs.TotalDataBytes) /
			timeStop.Sub(timeStart).Seconds())
		fmt.Printf("%s/s\n", format.FormatBytes(bytesPerSecond))
		if prev_fs != nil {
			if !scanner.CompareFileSystems(prev_fs, fs, os.Stdout) {
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
			w := bufio.NewWriter(file)
			fs.DebugWrite(w)
			w.Flush()
			file.Close()
		}
		if *gobFile != "" {
			file, err := os.Create(*gobFile)
			if err != nil {
				fmt.Printf("Error creating: %s\t%s\n", *gobFile, err)
				os.Exit(1)
			}
			encoder := gob.NewEncoder(file)
			encoderStartTime := time.Now()
			encoder.Encode(fs)
			fmt.Printf("Encoder time: %s\n", time.Since(encoderStartTime))
			file.Close()
		}
		prev_fs = fs
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}
