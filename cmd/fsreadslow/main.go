package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/lib/fsrateio"
)

// Benchmark the read speed of the underlying block device for a given file.
func main() {
	pathname := "/"
	if len(os.Args) == 2 {
		pathname = os.Args[1]
	}
	bytesPerSecond, blocksPerSecond, err := fsbench.GetReadSpeed(pathname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error! %s\n", err)
		return
	}
	ctx := fsrateio.NewReaderContext(bytesPerSecond, blocksPerSecond, 0)
	fmt.Println(ctx)
	var file *os.File
	file, err = os.Open(pathname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error! %s\n", err)
		return
	}
	rd := bufio.NewReader(ctx.NewReader(file))
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
			fmt.Fprintf(os.Stderr, "Error! %s\n", err)
			return
		}
		tread += n
	}
	bytesPerSecond = uint64(float64(tread) / time.Since(timeStart).Seconds())
	fmt.Printf("%s/s\n", format.FormatBytes(bytesPerSecond))
}
