package main

// Benchmark the scanning of a file-system tree.

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"io"
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
)

func writeNamedStat(writer io.Writer, name string, value uint64) {
	fmt.Fprintf(writer, "  %s=%s\n", name, fsrateio.FormatBytes(value))
}

func writeMemoryStats(writer io.Writer) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Fprintln(writer, "MemStats:")
	writeNamedStat(writer, "Alloc", memStats.Alloc)
	writeNamedStat(writer, "TotalAlloc", memStats.TotalAlloc)
	writeNamedStat(writer, "Sys", memStats.Sys)
}

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
		fs, err := scanner.ScanFileSystem(*rootDir, *objectCache, ctx)
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
		runtime.GC() // Clean up before showing memory statistics.
		writeMemoryStats(os.Stdout)
		if *debugFile != "" {
			file, err := os.Create(*debugFile)
			if err != nil {
				fmt.Printf("Error! %s\n", err)
				return
			}
			fs.DebugWrite(bufio.NewWriter(file), "")
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
