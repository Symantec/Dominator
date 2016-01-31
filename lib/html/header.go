package html

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"io"
	"runtime"
	"syscall"
	"time"
)

var (
	timeFormat string = "02 Jan 2006 15:04:05.99 MST"

	startTime  time.Time
	startUtime time.Time
	startStime time.Time
)

func init() {
	startTime = time.Now()
	startUtime, startStime = getRusage()
}

func getRusage() (time.Time, time.Time) {
	var rusage syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	return time.Unix(int64(rusage.Utime.Sec), int64(rusage.Utime.Usec)*1000),
		time.Unix(int64(rusage.Stime.Sec), int64(rusage.Stime.Usec)*1000)
}

func writeHeader(writer io.Writer) {
	fmt.Fprintf(writer, "Start time: %s<br>\n", startTime.Format(timeFormat))
	uptime := time.Since(startTime) + time.Millisecond*50
	uptime = (uptime / time.Millisecond / 100) * time.Millisecond * 100
	fmt.Fprintf(writer, "Uptime: %s<br>\n", uptime)
	uTime, sTime := getRusage()
	userCpuTime := uTime.Sub(startUtime)
	sysCpuTime := sTime.Sub(startStime)
	cpuTime := userCpuTime + sysCpuTime
	fmt.Fprintf(writer, "CPU Time: %.1f%% (User: %s Sys: %s)<br>\n",
		float64(cpuTime*100)/float64(uptime), userCpuTime, sysCpuTime)
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
