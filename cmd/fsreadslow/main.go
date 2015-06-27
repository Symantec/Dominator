package main

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"os"
	"time"
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
	ctx := fsrateio.NewContext(bytesPerSecond, blocksPerSecond)
	fmt.Println(ctx)
	var file *os.File
	file, err = os.Open(pathname)
	if err != nil {
		fmt.Printf("Error! %s\n", err)
		return
	}
	rd := bufio.NewReader(fsrateio.NewReader(file, ctx))
	buffer := make([]byte, 65536)
	timeStart := time.Now()
	tread := 0
	for {
		n := 0
		n, err = rd.Read(buffer)
		if n < 1 && err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			return
		}
		tread += n
	}
	bytesPerSecond = uint64(float64(tread) / time.Since(timeStart).Seconds())
	fmt.Printf("%s/s\n", fsrateio.FormatBytes(bytesPerSecond))
}
