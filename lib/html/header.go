package html

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"io"
	"runtime"
	"syscall"
	"time"
)

var timeFormat string = "02 Jan 2006 15:04:05.99 MST"

var startTime time.Time = time.Now()

func writeHeader(writer io.Writer) {
	fmt.Fprintf(writer, "Start time: %s<br>\n", startTime.Format(timeFormat))
	uptime := time.Since(startTime) + time.Millisecond*50
	uptime = (uptime / time.Millisecond / 100) * time.Millisecond * 100
	fmt.Fprintf(writer, "Uptime: %s<br>\n", uptime)
	var rusage syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	userCpuTime := time.Duration(rusage.Utime.Sec)*time.Second +
		time.Duration(rusage.Utime.Usec)*time.Microsecond
	sysCpuTime := time.Duration(rusage.Stime.Sec)*time.Second +
		time.Duration(rusage.Stime.Usec)*time.Microsecond
	cpuTime := rusage.Utime.Sec + rusage.Stime.Sec
	fmt.Fprintf(writer, "CPU Time: %.1f%% (User: %s Sys: %s)<br>\n",
		float64(cpuTime*100)/float64(uptime.Seconds()), userCpuTime, sysCpuTime)
	var memStatsBeforeGC, memStatsAfterGC runtime.MemStats
	runtime.ReadMemStats(&memStatsBeforeGC)
	runtime.GC()
	runtime.ReadMemStats(&memStatsAfterGC)
	fmt.Fprintf(writer, "Allocated memory: %s (%s after GC)<br>\n",
		format.FormatBytes(memStatsBeforeGC.Alloc),
		format.FormatBytes(memStatsAfterGC.Alloc))
	fmt.Fprintf(writer, "System memory: %s (%s after GC)<br>\n",
		format.FormatBytes(memStatsBeforeGC.Sys),
		format.FormatBytes(memStatsAfterGC.Sys))
	fmt.Fprintln(writer, "Raw <a href=\"metrics\">metrics</a>")
}
