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
	"github.com/Symantec/Dominator/lib/json"
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
		"Name of file to write GOB-encoded data to")
	interval = flag.Uint("interval", 0, "Seconds to sleep after each scan")
	jsonFile = flag.String("jsonFile", "",
		"Name of file to write JSON-encoded data to")
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
	var configuration scanner.Configuration
	configuration.ScanFilter, err = filter.New(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create empty filter: %s\n", err)
		os.Exit(1)
	}
	if *scanSpeed < 100 {
		bytesPerSecond, blocksPerSecond, err := fsbench.GetReadSpeed(*rootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error! %s\n", err)
			return
		}
		configuration.FsScanContext = fsrateio.NewReaderContext(bytesPerSecond,
			blocksPerSecond, 0)
		configuration.FsScanContext.GetContext().SetSpeedPercent(*scanSpeed)
		fmt.Println(configuration.FsScanContext)
	}
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
			fmt.Fprintf(os.Stderr, "Error! %s\n", err)
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
				fmt.Fprintf(os.Stderr, "Error! %s\n", err)
				return
			}
			w := bufio.NewWriter(file)
			fs.List(w)
			w.Flush()
			file.Close()
		}
		if *gobFile != "" {
			file, err := os.Create(*gobFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating: %s: %s\n",
					*gobFile, err)
				os.Exit(1)
			}
			encoder := gob.NewEncoder(file)
			encoderStartTime := time.Now()
			encoder.Encode(fs)
			fmt.Printf("Encoder time: %s\n", time.Since(encoderStartTime))
			file.Close()
		}
		if *jsonFile != "" {
			file, err := os.Create(*jsonFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating: %s: %s\n",
					*jsonFile, err)
				os.Exit(1)
			}
			if err := json.WriteWithIndent(file, "    ", fs); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing JSON: %s\n", err)
				os.Exit(1)
			}
			file.Close()
		}
		prev_fs = fs
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}
