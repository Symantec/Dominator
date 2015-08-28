package html

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"io"
	"runtime"
	"syscall"
	"time"
)

var startTime time.Time = time.Now()

func writeHeader(writer io.Writer) {
	fmt.Fprintf(writer, "Start time: %s<br>\n", startTime)
	uptime := time.Since(startTime)
	fmt.Fprintf(writer, "Uptime: %s<br>\n", uptime)
	var rusage syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	cpuTime := rusage.Utime.Sec + rusage.Stime.Sec
	fmt.Fprintf(writer, "CPU Time: %d%%<br>\n",
		cpuTime*100/int64(uptime.Seconds()))
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Fprintf(writer, "Allocated memory: %s<br>\n",
		format.FormatBytes(memStats.Alloc))
}
