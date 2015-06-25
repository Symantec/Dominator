package main

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"syscall"
	"time"
)

// Benchmark the scanning of a file-system tree.
func main() {
	pathname := "/"
	if len(os.Args) == 2 {
		pathname = os.Args[1]
	}
	ctx, err := fsrateio.NewContext(pathname)
	if err != nil {
		fmt.Printf("Error! %s\n", err)
		return
	}
	fmt.Println(ctx)
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
	timeStart := time.Now()
	fs, err := scanner.ScanFileSystem(pathname, ctx)
	timeStop := time.Now()
	if err != nil {
		fmt.Printf("Error! %s\n", err)
		return
	}
	fmt.Println(fs)
	var tread uint64 = 0
	for _, inode := range fs.InodeTable {
		tread += inode.Length()
	}
	fmt.Printf("Total scanned: %s\n", fsrateio.FormatBytes(tread))
	bytesPerSecond := uint64(float64(tread) / timeStop.Sub(timeStart).Seconds())
	fmt.Printf("%s/s\n", fsrateio.FormatBytes(bytesPerSecond))
}
