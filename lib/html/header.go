package html

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/format"
)

type cpuStats struct {
	realTime time.Time
	userTime time.Time
	sysTime  time.Time
}

var (
	startCpuStats *cpuStats = getCpuStats()
	lastCpuStats  *cpuStats = startCpuStats
)

func handleFunc(serveMux *http.ServeMux, pattern string,
	handler func(w http.ResponseWriter, req *http.Request)) {
	serveMux.HandleFunc(pattern,
		func(w http.ResponseWriter, req *http.Request) {
			SetSecurityHeaders(w) // Compliance checkbox.
			handler(w, req)
		})
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self' ;style-src 'self' 'unsafe-inline'")
}

func writeCpuStats(writer io.Writer, prefix string, start, current *cpuStats) {
	userCpuTime := current.userTime.Sub(start.userTime)
	sysCpuTime := current.sysTime.Sub(start.sysTime)
	realTime := current.realTime.Sub(start.realTime)
	cpuTime := userCpuTime + sysCpuTime
	fmt.Fprintf(writer,
		"    <td>%s CPU Time: %.1f%% (User: %s Sys: %s)</td>\n",
		prefix, float64(cpuTime*100)/float64(realTime), userCpuTime, sysCpuTime)
}

func writeHeader(writer io.Writer, req *http.Request, noGC bool) {
	currentCpuStats := getCpuStats()
	fmt.Fprintln(writer,
		`<table border="1" bordercolor=#e0e0e0 style="border-collapse: collapse">`)
	fmt.Fprintf(writer, "  <tr>\n")
	fmt.Fprintf(writer, "    <td>Start time: %s</td>\n",
		startCpuStats.realTime.Format(format.TimeFormatSeconds))
	uptime := currentCpuStats.realTime.Sub(startCpuStats.realTime)
	uptime += time.Millisecond * 50
	uptime = (uptime / time.Millisecond / 100) * time.Millisecond * 100
	fmt.Fprintf(writer, "    <td>Uptime: %s</td>\n", format.Duration(uptime))
	fmt.Fprintf(writer, "  </tr>\n")
	fmt.Fprintf(writer, "  <tr>\n")
	writeCpuStats(writer, "Total", startCpuStats, currentCpuStats)
	writeCpuStats(writer, "Recent", lastCpuStats, currentCpuStats)
	lastCpuStats = currentCpuStats
	fmt.Fprintf(writer, "  </tr>\n")
	fmt.Fprintf(writer, "  <tr>\n")
	var memStatsBeforeGC runtime.MemStats
	runtime.ReadMemStats(&memStatsBeforeGC)
	if noGC {
		fmt.Fprintf(writer, "    <td>Allocated memory: %s</td>\n",
			format.FormatBytes(memStatsBeforeGC.Alloc))
		fmt.Fprintf(writer, "    <td>System memory: %s</td>\n",
			format.FormatBytes(
				memStatsBeforeGC.Sys-memStatsBeforeGC.HeapReleased))
	} else {
		var memStatsAfterGC runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStatsAfterGC)
		fmt.Fprintf(writer, "    <td>Allocated memory: %s (%s after GC)</td>\n",
			format.FormatBytes(memStatsBeforeGC.Alloc),
			format.FormatBytes(memStatsAfterGC.Alloc))
		fmt.Fprintf(writer, "    <td>System memory: %s (%s after GC)</td>\n",
			format.FormatBytes(
				memStatsBeforeGC.Sys-memStatsBeforeGC.HeapReleased),
			format.FormatBytes(
				memStatsAfterGC.Sys-memStatsAfterGC.HeapReleased))
	}
	fmt.Fprintf(writer, "  </tr>\n")
	fmt.Fprintf(writer, "</table>\n")
	fmt.Fprintln(writer, "Raw <a href=\"metrics\">metrics</a><br>")
	if req != nil {
		protocol := "http"
		if req.TLS != nil {
			protocol = "https"
		}
		host := strings.Split(req.Host, ":")[0]
		fmt.Fprintf(writer,
			"Local <a href=\"%s://%s:6910/\">system health agent</a>",
			protocol, host)
	}
}

func getCpuStats() *cpuStats {
	uTime, sTime := getRusage()
	return &cpuStats{
		realTime: time.Now(),
		userTime: uTime,
		sysTime:  sTime,
	}
}

func getRusage() (time.Time, time.Time) {
	var rusage syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	return time.Unix(int64(rusage.Utime.Sec), int64(rusage.Utime.Usec)*1000),
		time.Unix(int64(rusage.Stime.Sec), int64(rusage.Stime.Usec)*1000)
}
